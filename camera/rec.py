#!/usr/bin/python3
"""Camera widget for screen recording.

The widget is an ordinary window, so the existing Omarchy recorder
(gpu-screen-recorder) captures it in place along with the microphone.
Keys 1 through 6 on the rec screen start, pause, resume, stop, focus, and quit.
"""

from __future__ import annotations

import argparse
import json
import math
import os
import shutil
import signal
import socket
import struct
import subprocess
import sys
import threading
import time
import warnings
from pathlib import Path

import cairo

import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Gdk", "4.0")
gi.require_version("GdkPixbuf", "2.0")
gi.require_version("GLibUnix", "2.0")
from gi.repository import Gdk, GdkPixbuf, Gio, GLib, GLibUnix, Gtk, Pango, PangoCairo

APP_ID = "com.ryan.rec"
MARGIN = 40
PILL_GAP = 8
PILL_H = 28
SIZES = {"small": 136, "medium": 176, "large": 220}
COMMANDS = ("open", "toggle", "start", "pause", "resume", "stop", "focus", "quit", "status")
HELP = """\
rec — record the screen with you in the corner, or filling the screen

Usage:
  rec                     Show the camera widget
  rec start               Start recording
  rec pause               Pause the recording
  rec resume              Resume the recording
  rec stop                Stop the recording and save it
  rec focus               Fill the recording with you. Press again for the computer
  rec quit                Save the recording, then close rec
  rec toggle              Start or stop a recording
  rec status              Print what the widget is doing
  rec --size small|medium|large

The rec screen lists:
  start
  pause
  resume
  stop and save, camera stays open
  fill the recording with you
  save and close
  recent recordings, newest first

j and k move. Enter runs the selected one. Enter on a recording plays it.
The camera stays on while it is open. Recordings include your voice
and are saved in Videos. --size is remembered and applies the next
time the camera opens.

rec runs r2d2 rec. On a terminal, r2d2 rec shows these controls.
With REC_INTERNAL=1, rec is the camera widget itself.
"""


def view_size(name: str) -> tuple[int, int]:
    diameter = SIZES[name]
    return diameter, diameter + PILL_GAP + PILL_H


def format_clock(seconds: float) -> str:
    total = max(0, int(seconds))
    minutes, secs = divmod(total, 60)
    hours, minutes = divmod(minutes, 60)
    if hours:
        return f"{hours}:{minutes:02d}:{secs:02d}"
    return f"{minutes}:{secs:02d}"


def elapsed_seconds(
    now: float,
    started_at: float | None,
    paused_total: float,
    pause_started: float | None,
) -> float:
    """Recording time with paused stretches left out."""
    if started_at is None:
        return 0.0
    held = paused_total
    if pause_started is not None:
        held += max(0.0, now - pause_started)
    return max(0.0, now - started_at - held)


def extract_jpegs(data: bytes, pending: bytes = b"") -> tuple[list[bytes], bytes]:
    buf = pending + data
    frames: list[bytes] = []
    while True:
        start = buf.find(b"\xff\xd8")
        if start < 0:
            return frames, buf[-1:] if buf.endswith(b"\xff") else b""
        end = buf.find(b"\xff\xd9", start + 2)
        if end < 0:
            return frames, buf[start:]
        frames.append(buf[start : end + 2])
        buf = buf[end + 2 :]


def rms_level(pcm: bytes) -> float:
    """Map signed 16-bit PCM to a 0..1 meter. Speech sits well below full scale."""
    count = len(pcm) // 2
    if count <= 0:
        return 0.0
    samples = struct.unpack(f"<{count}h", pcm[: count * 2])
    energy = sum(sample * sample for sample in samples) / count
    amplitude = math.sqrt(energy) / 32768
    return max(0.0, min(1.0, amplitude * 6))


def monitor_span(monitor: dict) -> tuple[int, int, int, int]:
    rotated = int(monitor.get("transform") or 0) % 2 == 1
    width = monitor["height"] if rotated else monitor["width"]
    height = monitor["width"] if rotated else monitor["height"]
    scale = float(monitor.get("scale") or 1)
    return (
        int(monitor["x"]),
        int(monitor["y"]),
        math.floor(width / scale),
        math.floor(height / scale),
    )


def corner(
    anchor_x: int,
    anchor_y: int,
    anchor_w: int,
    anchor_h: int,
    width: int,
    height: int,
    margin: int = MARGIN,
) -> tuple[int, int]:
    x = anchor_x + anchor_w - width - margin
    y = anchor_y + anchor_h - height - margin
    if x < anchor_x + margin:
        x = anchor_x + margin
    if y < anchor_y + margin:
        y = anchor_y + margin
    return x, y


def cover_rect(
    src_w: float,
    src_h: float,
    dst_w: float,
    dst_h: float,
) -> tuple[float, float, float, float]:
    """Destination rectangle that covers dst with src, centered."""
    if src_w <= 0 or src_h <= 0 or dst_w <= 0 or dst_h <= 0:
        return 0.0, 0.0, float(dst_w), float(dst_h)
    scale = max(dst_w / src_w, dst_h / src_h)
    width = src_w * scale
    height = src_h * scale
    return (dst_w - width) / 2, (dst_h - height) / 2, width, height


def capture_size(focused: bool) -> str:
    # The laptop camera delivers 1080p MJPG at 30 fps. The corner only needs a small frame.
    return "1920x1080" if focused else "640x360"


def covers_monitor(client: dict, monitor: dict) -> bool:
    _x, _y, width, height = monitor_span(monitor)
    size = client.get("size") or [0, 0]
    if len(size) < 2:
        return False
    return int(size[0]) >= width - 2 and int(size[1]) >= height - 2


def config_path() -> Path:
    # r2d2 rec owns this file: it writes --size here. The widget only reads it.
    return Path.home() / ".config" / "rec" / "size"


def load_size() -> str:
    try:
        name = config_path().read_text().strip()
    except OSError:
        return "small"
    return name if name in SIZES else "small"


def socket_path() -> Path:
    runtime = os.environ.get("XDG_RUNTIME_DIR") or f"/tmp/rec-{os.getuid()}"
    return Path(runtime) / "rec.sock"


def recording_active() -> bool:
    proc = subprocess.run(
        ["pgrep", "-f", "^gpu-screen-recorder"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    return proc.returncode == 0


def quit_needs_stop(mode: str, recorder_running: bool) -> bool:
    """Closing rec saves a take that is starting, running, or paused."""
    return mode in {"recording", "paused", "starting", "stopping", "saving"} or recorder_running


def recordings_dir() -> Path:
    override = os.environ.get("OMARCHY_SCREENRECORD_DIR")
    if override:
        return Path(override)
    configured = os.environ.get("XDG_VIDEOS_DIR")
    if configured:
        return Path(configured)
    dirs = Path.home() / ".config" / "user-dirs.dirs"
    try:
        for line in dirs.read_text().splitlines():
            if not line.startswith("XDG_VIDEOS_DIR="):
                continue
            value = line.split("=", 1)[1].strip().strip('"')
            return Path(os.path.expandvars(value))
    except OSError:
        pass
    return Path.home() / "Videos"


def recording_path_from_output(text: str) -> str:
    # Omarchy prints the saved path. A notification failure can follow it on stdout.
    lines = [line.strip() for line in text.splitlines() if line.strip()]
    files = [line for line in lines if line.endswith(".mp4")]
    for line in reversed(files):
        if Path(line).is_file():
            return line
    return files[-1] if files else ""


def recorded_filename() -> str:
    try:
        name = Path("/tmp/omarchy-screenrecord-filename").read_text().strip()
    except OSError:
        return ""
    return name if name.endswith(".mp4") else ""


def newest_recording_since(moment: float) -> str:
    directory = recordings_dir()
    if not directory.is_dir():
        return ""
    newest: Path | None = None
    newest_mtime = 0.0
    for path in directory.glob("screenrecording-*.mp4"):
        if "-processed" in path.name:
            continue
        try:
            mtime = path.stat().st_mtime
        except OSError:
            continue
        if mtime + 1 < moment or mtime < newest_mtime:
            continue
        newest = path
        newest_mtime = mtime
    return str(newest) if newest is not None else ""


def stop_active_recording() -> str:
    started = time.time()
    try:
        proc = subprocess.run(
            ["omarchy-capture-screenrecording", "--stop-recording"],
            capture_output=True,
            text=True,
            check=False,
        )
    except OSError as exc:
        print(exc, file=sys.stderr)
        return ""
    path = recording_path_from_output(proc.stdout or "")
    if not path or not Path(path).is_file():
        path = recorded_filename()
    if (not path or not Path(path).is_file()) and not recording_active():
        path = newest_recording_since(started)
    if not path or not Path(path).is_file():
        err = (proc.stderr or "").strip().splitlines()
        if err:
            print("\n".join(err[-12:]), file=sys.stderr)
        return ""
    return path


def signal_recorder(signum: int) -> bool:
    # Omarchy does not start gpu-screen-recorder with -ipc. SIGUSR2 toggles pause.
    name = signal.Signals(signum).name
    proc = subprocess.run(
        ["pkill", f"-{name}", "-f", "^gpu-screen-recorder"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    return proc.returncode == 0


def default_camera() -> str:
    try:
        proc = subprocess.run(
            ["omarchy-capture-webcam-list"],
            capture_output=True,
            text=True,
            check=False,
            timeout=5,
        )
    except (OSError, subprocess.TimeoutExpired):
        return "/dev/video0"
    for line in proc.stdout.splitlines():
        device = line.split()[0] if line.split() else ""
        if device.startswith("/dev/video"):
            return device
    return "/dev/video0"


def pixbuf_from_jpeg(data: bytes, side: int) -> GdkPixbuf.Pixbuf | None:
    pixbuf = pixbuf_from_jpeg_frame(data)
    if pixbuf is None or side < 1:
        return None
    width, height = pixbuf.get_width(), pixbuf.get_height()
    crop = min(width, height)
    if crop < 1:
        return None
    square = pixbuf.new_subpixbuf((width - crop) // 2, (height - crop) // 2, crop, crop)
    if square is None:
        return None
    if square.get_width() == side and square.get_height() == side:
        return square
    return square.scale_simple(side, side, GdkPixbuf.InterpType.BILINEAR)


def pixbuf_from_jpeg_frame(data: bytes) -> GdkPixbuf.Pixbuf | None:
    if not data:
        return None
    loader = GdkPixbuf.PixbufLoader.new_with_mime_type("image/jpeg")
    try:
        loader.write(data)
        loader.close()
        return loader.get_pixbuf()
    except Exception:
        return None


def _round_rect(ctx: cairo.Context, x: float, y: float, width: float, height: float, radius: float) -> None:
    ctx.new_sub_path()
    ctx.arc(x + width - radius, y + radius, radius, -math.pi / 2, 0)
    ctx.arc(x + width - radius, y + height - radius, radius, 0, math.pi / 2)
    ctx.arc(x + radius, y + height - radius, radius, math.pi / 2, math.pi)
    ctx.arc(x + radius, y + radius, radius, math.pi, 3 * math.pi / 2)
    ctx.close_path()


def _paint_pixbuf(ctx: cairo.Context, pixbuf, x: float, y: float, width: float, height: float) -> None:
    src_w = pixbuf.get_width()
    src_h = pixbuf.get_height()
    if src_w < 1 or src_h < 1:
        return
    ctx.save()
    ctx.translate(x, y)
    ctx.scale(width / src_w, height / src_h)
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", DeprecationWarning)
        Gdk.cairo_set_source_pixbuf(ctx, pixbuf, 0, 0)
        ctx.paint()
    ctx.restore()


def _paint_pill(
    ctx: cairo.Context,
    pill_x: float,
    pill_y: float,
    pill_w: float,
    label: str,
    show_dot: bool,
    paused: bool,
) -> None:
    _round_rect(ctx, pill_x, pill_y, pill_w, PILL_H, PILL_H / 2)
    ctx.set_source_rgba(0.07, 0.07, 0.07, 0.92)
    ctx.fill()

    layout = PangoCairo.create_layout(ctx)
    layout.set_font_description(Pango.FontDescription.from_string("Sans 11"))
    layout.set_text(label, -1)
    text_w, text_h = layout.get_pixel_size()
    if show_dot or paused:
        group = 8 + 6 + text_w
        left = pill_x + (pill_w - group) / 2
        ctx.arc(left + 4, pill_y + PILL_H / 2, 4, 0, math.tau)
        if paused:
            ctx.set_source_rgba(1, 0.74, 0.28, 1)
        else:
            ctx.set_source_rgba(0.93, 0.22, 0.18, 1)
        ctx.fill()
        text_x = left + 14
    else:
        text_x = pill_x + (pill_w - text_w) / 2
    ctx.set_source_rgba(1, 1, 1, 1)
    ctx.move_to(text_x, pill_y + (PILL_H - text_h) / 2)
    PangoCairo.show_layout(ctx, layout)


def paint(
    ctx: cairo.Context,
    diameter: int,
    pixbuf,
    pixbuf_scale: int,
    level: float,
    mode: str,
    label: str,
    show_dot: bool,
    camera_down: bool,
) -> None:
    ctx.set_operator(cairo.OPERATOR_SOURCE)
    ctx.set_source_rgba(0, 0, 0, 0)
    ctx.paint()
    ctx.set_operator(cairo.OPERATOR_OVER)

    cx = diameter / 2
    cy = diameter / 2
    radius = diameter / 2 - 4

    ctx.save()
    ctx.arc(cx, cy, radius - 1.5, 0, math.tau)
    ctx.clip()
    if pixbuf is not None:
        scale = pixbuf_scale or 1
        _paint_pixbuf(
            ctx,
            pixbuf,
            cx - (pixbuf.get_width() / scale) / 2,
            cy - (pixbuf.get_height() / scale) / 2,
            pixbuf.get_width() / scale,
            pixbuf.get_height() / scale,
        )
    else:
        ctx.set_source_rgba(0.10, 0.10, 0.10, 1)
        ctx.paint()
        if camera_down:
            _show_text(ctx, "No camera", diameter / 2, diameter / 2, 0, 1, 1, 1, 1)
    ctx.restore()

    ctx.set_line_width(5)
    ctx.set_source_rgba(0, 0, 0, 0.45)
    ctx.arc(cx, cy, radius, 0, math.tau)
    ctx.stroke()

    ctx.set_line_width(3)
    if mode == "recording":
        ctx.set_source_rgba(0.93, 0.22, 0.18, 1)
    elif mode == "paused":
        ctx.set_source_rgba(1, 0.74, 0.28, 1)
    else:
        ctx.set_source_rgba(1, 1, 1, 0.92)
    ctx.arc(cx, cy, radius, 0, math.tau)
    ctx.stroke()

    if level > 0.03 and mode != "paused":
        ctx.set_line_width(4)
        ctx.set_source_rgba(1, 0.74, 0.28, 0.95)
        ctx.arc(cx, cy, radius, -math.pi / 2, -math.pi / 2 + math.tau * min(level, 1))
        ctx.stroke()

    pill_w = diameter - 24
    _paint_pill(ctx, 12, diameter + PILL_GAP, pill_w, label, show_dot, mode == "paused")


def paint_cover(
    ctx: cairo.Context,
    width: int,
    height: int,
    pixbuf,
    mode: str,
    label: str,
    show_dot: bool,
    camera_down: bool,
) -> None:
    # Opaque on purpose: the screen recorder captures this window, so the desktop stays out of the file.
    ctx.set_operator(cairo.OPERATOR_SOURCE)
    ctx.set_source_rgba(0, 0, 0, 1)
    ctx.paint()
    ctx.set_operator(cairo.OPERATOR_OVER)
    if pixbuf is not None:
        x, y, draw_w, draw_h = cover_rect(pixbuf.get_width(), pixbuf.get_height(), width, height)
        _paint_pixbuf(ctx, pixbuf, x, y, draw_w, draw_h)
    elif camera_down:
        _show_text(ctx, "No camera", width / 2, height / 2, 0, 1, 1, 1, 1)
    pill_w = min(220, max(80, width - 48))
    _paint_pill(ctx, (width - pill_w) / 2, height - PILL_H - 28, pill_w, label, show_dot, mode == "paused")


def _show_text(ctx, text, cx, cy, _unused, red, green, blue, alpha) -> None:
    layout = PangoCairo.create_layout(ctx)
    layout.set_font_description(Pango.FontDescription.from_string("Sans 12"))
    layout.set_text(text, -1)
    text_w, text_h = layout.get_pixel_size()
    ctx.set_source_rgba(red, green, blue, alpha)
    ctx.move_to(cx - text_w / 2, cy - text_h / 2)
    PangoCairo.show_layout(ctx, layout)


def hypr_dispatch(expression: str) -> None:
    subprocess.run(
        ["hyprctl", "dispatch", expression],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=False,
    )


def fullscreen_expression(address: str, action: str) -> str:
    return (
        'hl.dsp.window.fullscreen({ window = "address:'
        + address
        + '", mode = "fullscreen", action = "'
        + action
        + '" })'
    )


def pin_expression(address: str, action: str) -> str:
    return 'hl.dsp.window.pin({ window = "address:' + address + '", action = "' + action + '" })'


def focused_monitor_span() -> tuple[int, int, int, int] | None:
    try:
        monitors = json.loads(subprocess.check_output(["hyprctl", "monitors", "-j"], text=True))
    except (OSError, subprocess.CalledProcessError, json.JSONDecodeError):
        return None
    if not monitors:
        return None
    monitor = next((item for item in monitors if item.get("focused")), monitors[0])
    return monitor_span(monitor)


def find_rec_window() -> tuple[dict, dict] | None:
    try:
        clients = json.loads(subprocess.check_output(["hyprctl", "clients", "-j"], text=True))
        monitors = json.loads(subprocess.check_output(["hyprctl", "monitors", "-j"], text=True))
    except (OSError, subprocess.CalledProcessError, json.JSONDecodeError):
        return None
    me = next((client for client in clients if client.get("class") == APP_ID), None)
    if me is None:
        return None
    monitor = next((item for item in monitors if item.get("id") == me.get("monitor")), None)
    if monitor is None:
        return None
    return me, monitor


def try_send(command: str, timeout: float) -> str | None:
    path = socket_path()
    if not path.exists():
        return None
    try:
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
            client.settimeout(timeout)
            client.connect(str(path))
            client.sendall((command + "\n").encode())
            data = b""
            while b"\n" not in data:
                chunk = client.recv(4096)
                if not chunk:
                    break
                data += chunk
    except (ConnectionRefusedError, FileNotFoundError, socket.timeout, OSError):
        return None
    return data.decode(errors="replace").strip()


def r2d2_argv(argv: list[str], env: dict[str, str] | os._Environ) -> list[str] | None:
    """The r2d2 rec command this rec hands to. None when rec is the widget itself."""
    if env.get("REC_INTERNAL") == "1":
        return None
    installed = Path.home() / ".local" / "bin" / "r2d2"
    program = str(installed) if os.access(installed, os.X_OK) else shutil.which("r2d2")
    if not program:
        return None
    return [program, "rec", *argv]


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(prog="rec", add_help=False)
    parser.add_argument("command", nargs="?", default="open", choices=COMMANDS)
    parser.add_argument("--size", choices=sorted(SIZES))
    parser.add_argument("--device", help="V4L2 camera node, such as /dev/video0")
    parser.add_argument("-h", "--help", action="store_true")
    return parser.parse_args(argv)


class RecWindow:
    def __init__(
        self,
        app: Gtk.Application,
        size_name: str,
        device: str,
        record_on_start: bool,
        focused: bool,
    ) -> None:
        self.app = app
        self.size_name = size_name
        self.device = device
        self.record_on_start = record_on_start
        self.focused = focused
        self.diameter = SIZES[size_name]
        self.mode = "idle"
        self.started_at: float | None = None
        self.pause_started: float | None = None
        self.paused_total = 0.0
        self.error = ""
        self.quit_after = False
        self.stop_thread: threading.Thread | None = None
        self.level = 0.0
        self.pixbuf = None
        self.pixbuf_scale = 1
        self.camera_down = False
        self.latest: bytes | None = None
        self.lock = threading.Lock()
        self.quit_event = threading.Event()
        self.camera_generation = 0
        self.ffmpeg: subprocess.Popen | None = None
        self.parec: subprocess.Popen | None = None
        self.pending_conn: socket.socket | None = None
        self.server: socket.socket | None = None

        width, height = view_size(size_name)
        if focused:
            span = focused_monitor_span()
            if span is not None:
                width, height = span[2], span[3]
        self.window = Gtk.ApplicationWindow(application=app, title="Rec")
        self.window.set_decorated(False)
        self.window.set_resizable(not focused)
        self.window.set_default_size(width, height)
        self.window.connect("close-request", self._close)

        self.area = Gtk.DrawingArea()
        self.area.set_hexpand(True)
        self.area.set_vexpand(True)
        self.area.set_content_width(width)
        self.area.set_content_height(height)
        self.area.set_draw_func(self._draw, None)
        self.area.set_focusable(True)
        self.area.set_cursor_from_name("pointer")
        self.area.set_tooltip_text("The rec screen lists start, pause, resume, stop, you, and quit.")
        self.window.set_child(self.area)

        click = Gtk.GestureClick()
        click.set_button(0)
        click.connect("released", self._click)
        self.area.add_controller(click)
        keys = Gtk.EventControllerKey()
        keys.connect("key-pressed", self._key)
        self.area.add_controller(keys)

        css = Gtk.CssProvider()
        css.load_from_string("window, drawingarea { background: none; background-color: transparent; }")
        Gtk.StyleContext.add_provider_for_display(
            Gdk.Display.get_default(),
            css,
            Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
        )

        self._listen()
        threading.Thread(target=self._camera_loop, name="rec-camera", daemon=True).start()
        threading.Thread(target=self._audio_loop, name="rec-audio", daemon=True).start()
        GLib.timeout_add(100, self._tick)
        GLib.timeout_add(50, self._place_later, 40)
        GLibUnix.signal_add(GLib.PRIORITY_DEFAULT, signal.SIGTERM, self._signal_quit)
        GLibUnix.signal_add(GLib.PRIORITY_DEFAULT, signal.SIGINT, self._signal_quit)
        if record_on_start:
            GLib.timeout_add(700, self._record_once)

    def present(self) -> None:
        self.window.present()

    def _record_once(self) -> bool:
        if self.mode == "idle":
            self._begin_start()
        return False

    def _signal_quit(self) -> bool:
        self.request_quit()
        # Stay installed. Removing it lets a later signal kill the process mid-save.
        return True

    def _close(self, _window) -> bool:
        self.request_quit()
        return True

    def _click(self, gesture, _n_press, _x, _y) -> None:
        button = gesture.get_current_button()
        if button == Gdk.BUTTON_SECONDARY:
            self.request_quit()
        elif button == Gdk.BUTTON_PRIMARY:
            self.toggle()

    def _key(self, _controller, keyval, _keycode, _state) -> bool:
        if keyval == Gdk.KEY_Escape:
            self.request_quit()
            return True
        return False

    def _draw(self, _area, ctx, width, height, _data) -> None:
        label, show_dot = self._label()
        if self.focused:
            paint_cover(
                ctx,
                int(width),
                int(height),
                self.pixbuf,
                self.mode,
                label,
                show_dot,
                self.camera_down and self.pixbuf is None,
            )
            return
        paint(
            ctx,
            self.diameter,
            self.pixbuf,
            self.pixbuf_scale,
            self.level,
            self.mode,
            label,
            show_dot,
            self.camera_down and self.pixbuf is None,
        )

    def _elapsed(self) -> float:
        return elapsed_seconds(time.monotonic(), self.started_at, self.paused_total, self.pause_started)

    def _label(self) -> tuple[str, bool]:
        if self.mode == "paused":
            return "Paused", False
        if self.mode == "recording" and self.started_at is not None:
            return format_clock(self._elapsed()), True
        if self.mode == "saving":
            return "Saving", False
        if self.mode == "stopping":
            return "Stopping", True
        if self.mode == "starting":
            return "Starting", False
        if self.error:
            return self.error, False
        return "Record", True

    def _clear_timing(self) -> None:
        self.started_at = None
        self.pause_started = None
        self.paused_total = 0.0

    def _tick(self) -> bool:
        if self.quit_event.is_set():
            return False
        with self.lock:
            jpeg = self.latest
            self.latest = None
        if jpeg:
            scale = self.area.get_scale_factor() or 1
            if self.focused:
                pixbuf = pixbuf_from_jpeg_frame(jpeg)
                pixbuf_scale = 1
            else:
                pixbuf = pixbuf_from_jpeg(jpeg, self.diameter * scale)
                pixbuf_scale = scale
            if pixbuf is not None:
                self.pixbuf = pixbuf
                self.pixbuf_scale = pixbuf_scale
                self.camera_down = False
        active = recording_active()
        if self.mode == "paused":
            if not active:
                self.mode = "idle"
                self._clear_timing()
        elif self.mode == "idle" and active:
            self.mode = "recording"
            self.started_at = time.monotonic()
            self.error = ""
        elif self.mode == "recording" and not active:
            self.mode = "idle"
            self._clear_timing()
        elif self.mode == "stopping" and not active:
            self.mode = "saving"
        elif self.mode == "starting" and active:
            self.mode = "recording"
            self.started_at = self.started_at or time.monotonic()
            self.error = ""
        self.area.queue_draw()
        return True

    def _place_later(self, tries: int) -> bool:
        if self._apply_layout():
            return False
        if tries <= 1:
            return False
        GLib.timeout_add(50, self._place_later, tries - 1)
        return False

    def _apply_layout(self) -> bool:
        found = find_rec_window()
        if found is None:
            return False
        me, monitor = found
        address = str(me["address"])
        if self.focused:
            # A pinned window stays at the corner. Fullscreen applies after unpin.
            self.window.set_resizable(True)
            _ax, _ay, width, height = monitor_span(monitor)
            self.area.set_content_width(width)
            self.area.set_content_height(height)
            self.window.set_default_size(width, height)
            if me.get("pinned"):
                hypr_dispatch(pin_expression(address, "unset"))
            hypr_dispatch(fullscreen_expression(address, "set"))
            return int(me.get("fullscreen") or 0) == 2
        hypr_dispatch(fullscreen_expression(address, "unset"))
        width, height = view_size(self.size_name)
        self.window.set_resizable(False)
        self.window.set_default_size(width, height)
        self.area.set_content_width(width)
        self.area.set_content_height(height)
        if int(me.get("fullscreen") or 0) != 0:
            return False
        anchor_x, anchor_y, anchor_w, anchor_h = monitor_span(monitor)
        x, y = corner(anchor_x, anchor_y, anchor_w, anchor_h, width, height)
        window = f"address:{address}"
        if not me.get("pinned"):
            hypr_dispatch(pin_expression(address, "set"))
        hypr_dispatch(f'hl.dsp.window.resize({{ window = "{window}", x = {width}, y = {height} }})')
        hypr_dispatch(f'hl.dsp.window.move({{ window = "{window}", x = {x}, y = {y} }})')
        at = me.get("at") or [0, 0]
        size = me.get("size") or [0, 0]
        return (
            abs(int(at[0]) - x) <= 2
            and abs(int(at[1]) - y) <= 2
            and abs(int(size[0]) - width) <= 2
            and abs(int(size[1]) - height) <= 2
        )

    def _listen(self) -> None:
        path = socket_path()
        path.parent.mkdir(parents=True, exist_ok=True)
        try:
            path.unlink()
        except FileNotFoundError:
            pass
        server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        server.bind(str(path))
        os.chmod(path, 0o600)
        server.listen(4)
        server.setblocking(False)
        self.server = server
        GLib.io_add_watch(server.fileno(), GLib.IO_IN, self._accept)

    def _accept(self, _fd, _condition) -> bool:
        if self.server is None:
            return False
        try:
            conn, _addr = self.server.accept()
        except BlockingIOError:
            return True
        conn.settimeout(2)
        try:
            raw = b""
            while b"\n" not in raw and len(raw) < 256:
                chunk = conn.recv(256)
                if not chunk:
                    break
                raw += chunk
            command = raw.decode(errors="replace").strip()
        except OSError:
            conn.close()
            return True
        self._dispatch_command(command, conn)
        return True

    def _dispatch_command(self, command: str, conn: socket.socket) -> None:
        if command == "status":
            self._reply(conn, self.status_line())
            return
        if command == "raise":
            self._apply_layout()
            self.present()
            self._reply(conn, self.status_line())
            return
        if command == "quit":
            self.request_quit(conn)
            return
        if command == "focus":
            self._reply(conn, self.toggle_focus())
            return
        if command == "pause":
            self._reply(conn, self.pause())
            return
        if command == "resume":
            self._reply(conn, self.resume())
            return
        if command == "start":
            if self.mode in {"recording", "paused"}:
                self._reply(conn, self.status_line())
                return
            if self.pending_conn is not None or self.mode in {"starting", "stopping", "saving"}:
                self._reply(conn, "busy")
                return
            self.pending_conn = conn
            self._begin_start()
            return
        if command not in {"toggle", "stop"}:
            self._reply(conn, "unknown")
            return
        if self.pending_conn is not None or self.mode in {"starting", "stopping", "saving"}:
            self._reply(conn, "busy")
            return
        self.pending_conn = conn
        if command == "stop":
            if self.mode in {"recording", "paused"} or recording_active():
                self._begin_stop()
            else:
                self._finish_reply("idle")
            return
        self.toggle()

    def _reply(self, conn: socket.socket, text: str) -> None:
        try:
            conn.sendall((text + "\n").encode())
        except OSError:
            pass
        finally:
            conn.close()

    def _finish_reply(self, text: str) -> None:
        conn = self.pending_conn
        self.pending_conn = None
        if conn is not None:
            self._reply(conn, text)

    def status_line(self) -> str:
        focus = " focus" if self.focused else ""
        if self.mode == "recording" and self.started_at is not None:
            return "recording " + format_clock(self._elapsed()) + focus
        if self.mode == "paused" and self.started_at is not None:
            return "paused " + format_clock(self._elapsed()) + focus
        if self.mode in {"starting", "stopping", "saving"}:
            return self.mode + focus
        if self.error:
            return "idle " + self.error
        if self.focused:
            return "idle focus"
        return "idle"

    def toggle_focus(self) -> str:
        self.focused = not self.focused
        self.camera_generation += 1
        with self.lock:
            self.latest = None
        self.pixbuf = None
        proc = self.ffmpeg
        if proc is not None and proc.poll() is None:
            proc.terminate()
        self._apply_layout()
        GLib.timeout_add(50, self._place_later, 30)
        self.area.queue_draw()
        return self.status_line()

    def pause(self) -> str:
        if self.mode != "recording":
            return self.status_line()
        if not signal_recorder(signal.SIGUSR2):
            self.error = "Could not pause"
            return self.status_line()
        self.mode = "paused"
        self.pause_started = time.monotonic()
        self.error = ""
        self.area.queue_draw()
        return self.status_line()

    def resume(self) -> str:
        if self.mode != "paused":
            return self.status_line()
        if not signal_recorder(signal.SIGUSR2):
            self.error = "Could not resume"
            return self.status_line()
        now = time.monotonic()
        if self.pause_started is not None:
            self.paused_total += max(0.0, now - self.pause_started)
        self.pause_started = None
        self.mode = "recording"
        self.error = ""
        self.area.queue_draw()
        return self.status_line()

    def toggle(self) -> None:
        if self.mode in {"starting", "stopping", "saving"}:
            return
        if self.mode in {"recording", "paused"} or recording_active():
            self._begin_stop()
            return
        self._begin_start()

    def _begin_start(self) -> None:
        if self.mode in {"recording", "paused"}:
            self._finish_reply(self.status_line())
            return
        if recording_active():
            self.mode = "recording"
            self.started_at = self.started_at or time.monotonic()
            self.error = ""
            self._finish_reply(self.status_line())
            return
        self.mode = "starting"
        self.error = ""
        self._clear_timing()
        self._apply_layout()
        threading.Thread(target=self._start_worker, name="rec-start", daemon=True).start()

    def _begin_stop(self) -> None:
        if self.stop_thread is not None and self.stop_thread.is_alive():
            self.mode = "stopping"
            return
        self.mode = "stopping"
        # A daemon thread dies when the window closes, and the file dies with it.
        self.stop_thread = threading.Thread(target=self._stop_worker, name="rec-stop", daemon=False)
        self.stop_thread.start()

    def _start_worker(self) -> None:
        # The picture has to be still before capture, or the face is recorded sliding in.
        time.sleep(0.35)
        env = os.environ.copy()
        env["OMARCHY_SCREENRECORD_DEBUG"] = "true"
        # The recorder keeps the script stdout open after the script exits.
        # A pipe here would not close until the recording stops.
        try:
            subprocess.run(
                ["omarchy-capture-screenrecording", "--fullscreen", "--with-microphone-audio"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                env=env,
                check=False,
                # The recorder must outlive rec. Sharing a process group lets close signal it first.
                start_new_session=True,
            )
            detail = ""
        except OSError as exc:
            detail = str(exc)
        active = recording_active()
        GLib.idle_add(self._start_finished, active, detail)

    def _start_finished(self, active: bool, detail: str) -> bool:
        if self.quit_event.is_set():
            return False
        if active:
            self.mode = "recording"
            self.started_at = self.started_at or time.monotonic()
            self.pause_started = None
            self.error = ""
            reply = self.status_line()
        else:
            self.mode = "idle"
            self._clear_timing()
            self.error = "Could not start"
            reply = "idle Could not start"
            if detail:
                print(detail, file=sys.stderr)
            log = Path("/tmp/omarchy-screenrecord.log")
            if log.exists():
                tail = log.read_text(errors="replace").splitlines()[-20:]
                if tail:
                    print("\n".join(tail), file=sys.stderr)
        if self.quit_after:
            if active or recording_active():
                self._begin_stop()
            else:
                self._shutdown()
        else:
            self._finish_reply(reply)
        self.area.queue_draw()
        return False

    def _stop_worker(self) -> None:
        path = ""
        try:
            path = stop_active_recording()
        except Exception as exc:
            print(exc, file=sys.stderr)
        GLib.idle_add(self._stop_finished, path)

    def _stop_finished(self, path: str) -> bool:
        self.mode = "idle"
        self._clear_timing()
        self.error = ""
        reply = path or "idle"
        self._finish_reply(reply)
        if self.quit_after or self.quit_event.is_set():
            self._shutdown()
            return False
        self.area.queue_draw()
        return False

    def _hold_quit_reply(self, conn: socket.socket | None) -> None:
        if conn is None:
            return
        if self.pending_conn is None and quit_needs_stop(self.mode, recording_active()):
            conn.settimeout(None)
            self.pending_conn = conn
            return
        self._reply(conn, "bye")

    def request_quit(self, conn: socket.socket | None = None) -> None:
        if self.quit_after and self.mode in {"stopping", "saving"}:
            self._hold_quit_reply(conn)
            return
        self.quit_after = True
        self._hold_quit_reply(conn)
        if self.mode in {"recording", "paused"} or (self.mode == "idle" and recording_active()):
            self._begin_stop()
            return
        if self.mode == "starting":
            return
        if self.mode in {"stopping", "saving"}:
            return
        if recording_active():
            self._begin_stop()
            return
        self._shutdown()

    def _shutdown(self) -> bool:
        if self.stop_thread is not None and self.stop_thread.is_alive():
            GLib.timeout_add(100, self._shutdown)
            return False
        if self.quit_event.is_set() and self.server is None:
            return False
        self.quit_event.set()
        for proc in (self.ffmpeg, self.parec):
            if proc is not None and proc.poll() is None:
                proc.terminate()
        if self.server is not None:
            try:
                self.server.close()
            except OSError:
                pass
            self.server = None
        try:
            socket_path().unlink()
        except FileNotFoundError:
            pass
        self._finish_reply("bye")
        self.app.quit()
        return False

    def _camera_loop(self) -> None:
        while not self.quit_event.is_set():
            generation = self.camera_generation
            size = capture_size(self.focused)
            proc = subprocess.Popen(
                [
                    "ffmpeg",
                    "-hide_banner",
                    "-loglevel",
                    "error",
                    "-fflags",
                    "nobuffer",
                    "-flags",
                    "low_delay",
                    "-f",
                    "v4l2",
                    "-input_format",
                    "mjpeg",
                    "-framerate",
                    "30",
                    "-video_size",
                    size,
                    "-i",
                    self.device,
                    "-an",
                    "-vf",
                    "fps=20",
                    "-f",
                    "image2pipe",
                    "-c:v",
                    "mjpeg",
                    "-q:v",
                    "5",
                    "pipe:1",
                ],
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                stdin=subprocess.DEVNULL,
                bufsize=0,
            )
            self.ffmpeg = proc
            if generation != self.camera_generation or self.quit_event.is_set():
                proc.terminate()
                continue
            pending = b""
            saw_frame = False
            assert proc.stdout is not None
            while not self.quit_event.is_set():
                chunk = proc.stdout.read(65536)
                if not chunk:
                    break
                if generation != self.camera_generation:
                    break
                frames, pending = extract_jpegs(chunk, pending)
                if frames:
                    saw_frame = True
                    with self.lock:
                        self.latest = frames[-1]
            if proc.poll() is None:
                proc.terminate()
                try:
                    proc.wait(timeout=1)
                except subprocess.TimeoutExpired:
                    proc.kill()
            if self.quit_event.is_set():
                break
            if generation != self.camera_generation:
                continue
            if not saw_frame:
                self.camera_down = True
                time.sleep(2)

    def _audio_loop(self) -> None:
        while not self.quit_event.is_set():
            proc = subprocess.Popen(
                [
                    "parec",
                    "--device",
                    "@DEFAULT_SOURCE@",
                    "--format=s16le",
                    "--rate=16000",
                    "--channels=1",
                    "--latency-msec=40",
                ],
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                stdin=subprocess.DEVNULL,
                bufsize=0,
            )
            self.parec = proc
            assert proc.stdout is not None
            while not self.quit_event.is_set():
                pcm = proc.stdout.read(1280)
                if not pcm:
                    break
                instant = rms_level(pcm)
                self.level = self.level * 0.55 + instant * 0.45
            if proc.poll() is None:
                proc.terminate()
            if self.quit_event.is_set():
                break
            self.level = 0.0
            time.sleep(2)


def run(size_name: str, device: str | None, record_on_start: bool, focused: bool) -> int:
    device = device or default_camera()
    app = Gtk.Application(application_id=APP_ID, flags=Gio.ApplicationFlags.NON_UNIQUE)
    holder: dict[str, RecWindow] = {}

    def activate(_app: Gtk.Application) -> None:
        if "window" not in holder:
            holder["window"] = RecWindow(app, size_name, device, record_on_start, focused)
        holder["window"].present()

    app.connect("activate", activate)
    return app.run(None)


def main(argv: list[str] | None = None) -> int:
    argv = sys.argv[1:] if argv is None else argv
    # r2d2 starts the widget with REC_INTERNAL=1, so this does not loop.
    handoff = r2d2_argv(argv, os.environ)
    if handoff is not None:
        os.execv(handoff[0], handoff)
    args = parse_args(argv)
    if args.help:
        print(HELP, end="")
        return 0
    size_name = args.size or load_size()
    if args.command == "open":
        command = "raise"
        timeout = 2
    elif args.command in {"pause", "resume", "focus", "status"}:
        command = args.command
        timeout = 2
    elif args.command in {"toggle", "start", "stop", "quit"}:
        command = args.command
        timeout = 600
    else:
        command = "status"
        timeout = 2
    reply = try_send(command, timeout)
    if reply is not None:
        if args.command != "open" and reply:
            print(reply)
        return 0
    if args.command == "quit" and recording_active():
        path = stop_active_recording()
        if path:
            print(path)
            return 0
    if args.command in {"stop", "quit", "status", "pause", "resume"}:
        print("rec is not open", file=sys.stderr)
        return 1
    return run(
        size_name,
        args.device,
        record_on_start=args.command in {"toggle", "start"},
        focused=args.command == "focus",
    )


if __name__ == "__main__":
    raise SystemExit(main())
