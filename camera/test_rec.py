#!/usr/bin/python3
"""Unit tests for rec geometry, jpeg splitting, and the meter."""

import os
import stat
import subprocess
import sys
import tempfile
import time
import unittest
import unittest.mock
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import rec


class RecTests(unittest.TestCase):
    def test_view_size_includes_the_pill(self):
        self.assertEqual(rec.view_size("small"), (136, 172))
        self.assertEqual(rec.view_size("medium"), (176, 212))
        self.assertEqual(rec.view_size("large"), (220, 256))

    def test_corner_sits_bottom_right_with_the_omarchy_margin(self):
        self.assertEqual(rec.corner(0, 0, 1280, 720, 136, 172), (1104, 508))

    def test_corner_stays_inside_a_tiny_anchor(self):
        self.assertEqual(rec.corner(10, 20, 50, 50, 136, 172), (50, 60))

    def test_monitor_span_uses_logical_pixels(self):
        span = rec.monitor_span(
            {"x": 0, "y": 0, "width": 1920, "height": 1080, "scale": 1.5, "transform": 0}
        )
        self.assertEqual(span, (0, 0, 1280, 720))

    def test_clock(self):
        self.assertEqual(rec.format_clock(0), "0:00")
        self.assertEqual(rec.format_clock(65), "1:05")
        self.assertEqual(rec.format_clock(3661), "1:01:01")

    def test_elapsed_seconds_omits_time_spent_paused(self):
        self.assertEqual(rec.elapsed_seconds(10, None, 0, None), 0.0)
        self.assertEqual(rec.elapsed_seconds(10, 0, 0, None), 10.0)
        self.assertEqual(rec.elapsed_seconds(10, 0, 0, 4), 4.0)
        self.assertEqual(rec.elapsed_seconds(10, 0, 3, 8), 5.0)

    def test_cover_rect_fills_the_destination(self):
        self.assertEqual(rec.cover_rect(1920, 1080, 1280, 720), (0.0, 0.0, 1280.0, 720.0))
        self.assertEqual(rec.cover_rect(1, 1, 2, 1), (0.0, -0.5, 2.0, 2.0))

    def test_capture_size_uses_the_full_camera_frame_when_focused(self):
        self.assertEqual(rec.capture_size(False), "640x360")
        self.assertEqual(rec.capture_size(True), "1920x1080")

    def test_covers_monitor_accepts_the_full_logical_size(self):
        monitor = {"x": 0, "y": 0, "width": 1920, "height": 1080, "scale": 1.5, "transform": 0}
        self.assertTrue(rec.covers_monitor({"size": [1280, 720]}, monitor))
        self.assertFalse(rec.covers_monitor({"size": [136, 172]}, monitor))

    def test_extract_jpegs_across_chunks(self):
        frame = b"\xff\xd8" + b"abc" + b"\xff\xd9"
        frames, pending = rec.extract_jpegs(frame[:3], b"")
        self.assertEqual(frames, [])
        self.assertEqual(pending, frame[:3])
        frames, pending = rec.extract_jpegs(frame[3:], pending)
        self.assertEqual(frames, [frame])
        self.assertEqual(pending, b"")

    def test_extract_jpegs_ignores_leading_noise_and_keeps_a_split_marker(self):
        frame = b"\xff\xd8hello\xff\xd9"
        frames, pending = rec.extract_jpegs(b"xx" + frame + frame + b"\xff", b"")
        self.assertEqual(frames, [frame, frame])
        self.assertEqual(pending, b"\xff")

    def test_rms_level_is_silent_at_zero_and_full_at_a_hot_signal(self):
        self.assertEqual(rec.rms_level(b""), 0.0)
        self.assertEqual(rec.rms_level(b"\x00\x00" * 16), 0.0)
        hot = struct_hot()
        self.assertEqual(rec.rms_level(hot), 1.0)

    def test_parse_args_defaults_to_open(self):
        args = rec.parse_args([])
        self.assertEqual(args.command, "open")
        self.assertIsNone(args.size)
        args = rec.parse_args(["--size", "large", "toggle"])
        self.assertEqual(args.command, "toggle")
        self.assertEqual(args.size, "large")

    def test_parse_args_accepts_the_keyboard_commands(self):
        for command in ("start", "pause", "resume", "stop", "focus", "quit"):
            self.assertEqual(rec.parse_args([command]).command, command)

    def test_quit_saves_a_running_take(self):
        self.assertTrue(rec.quit_needs_stop("recording", False))
        self.assertTrue(rec.quit_needs_stop("paused", False))
        self.assertTrue(rec.quit_needs_stop("starting", False))
        self.assertTrue(rec.quit_needs_stop("stopping", False))
        self.assertTrue(rec.quit_needs_stop("idle", True))
        self.assertFalse(rec.quit_needs_stop("idle", False))

    def test_recording_path_keeps_the_mp4_when_a_notification_error_follows(self):
        saved = "/home/ryan/Videos/screenrecording-2026-10-24_14-14-42.mp4"
        text = "\n".join(
            [
                "/tmp/missing-rec-test.mp4",
                saved,
                "Call failed: The name is not activatable",
            ]
        )
        self.assertEqual(rec.recording_path_from_output(text), saved)
        self.assertEqual(rec.recording_path_from_output("Call failed: no\n"), "")

    def test_newest_recording_since_skips_older_takes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            old = root / "screenrecording-old.mp4"
            new = root / "screenrecording-new.mp4"
            processed = root / "screenrecording-new-processed.mp4"
            old.write_bytes(b"old")
            new.write_bytes(b"new")
            processed.write_bytes(b"no")
            os.utime(old, (1, 1))
            now = time.time()
            os.utime(new, (now, now))
            os.utime(processed, (now, now))
            previous = os.environ.get("OMARCHY_SCREENRECORD_DIR")
            os.environ["OMARCHY_SCREENRECORD_DIR"] = tmp
            try:
                self.assertEqual(rec.newest_recording_since(now - 5), str(new))
                self.assertEqual(rec.newest_recording_since(now + 10), "")
            finally:
                if previous is None:
                    os.environ.pop("OMARCHY_SCREENRECORD_DIR", None)
                else:
                    os.environ["OMARCHY_SCREENRECORD_DIR"] = previous

    def test_config_path_is_the_r2d2_rec_size(self):
        with tempfile.TemporaryDirectory() as home:
            env = {"HOME": home, "XDG_CONFIG_HOME": ""}
            with unittest.mock.patch.dict(os.environ, env):
                self.assertEqual(rec.config_path(), Path(home) / ".config" / "r2d2" / "rec" / "size")
            env["XDG_CONFIG_HOME"] = "relative"
            with unittest.mock.patch.dict(os.environ, env):
                self.assertEqual(rec.config_path(), Path(home) / ".config" / "r2d2" / "rec" / "size")
            env["XDG_CONFIG_HOME"] = "/xdg"
            with unittest.mock.patch.dict(os.environ, env):
                self.assertEqual(rec.config_path(), Path("/xdg/r2d2/rec/size"))

    def test_load_size_reads_only_the_r2d2_file(self):
        with tempfile.TemporaryDirectory() as home:
            old = Path(home) / ".config" / "rec" / "size"
            old.parent.mkdir(parents=True)
            old.write_text("large\n")
            with unittest.mock.patch.dict(os.environ, {"HOME": home, "XDG_CONFIG_HOME": ""}):
                self.assertEqual(rec.load_size(), "small")
                new = rec.config_path()
                new.parent.mkdir(parents=True)
                new.write_text("medium\n")
                self.assertEqual(rec.load_size(), "medium")

    def test_rec_internal_does_not_hand_off_to_r2d2(self):
        self.assertIsNone(rec.r2d2_argv(["start"], {"REC_INTERNAL": "1"}))

    def test_rec_hands_off_to_the_installed_r2d2(self):
        with tempfile.TemporaryDirectory() as tmp:
            r2d2 = fake_program(Path(tmp) / ".local" / "bin" / "r2d2", "")
            with home(tmp):
                self.assertEqual(rec.r2d2_argv(["start"], {}), [str(r2d2), "rec", "start"])
                self.assertEqual(rec.r2d2_argv([], {}), [str(r2d2), "rec"])

    def test_install_writes_a_launcher_that_runs_r2d2_unless_rec_internal(self):
        here = Path(__file__).resolve().parent
        with tempfile.TemporaryDirectory() as tmp:
            env = dict(os.environ, HOME=tmp)
            env.pop("REC_INTERNAL", None)
            subprocess.run(["sh", str(here / "install.sh")], env=env, check=True, cwd=tmp)
            launcher = Path(tmp) / ".local" / "bin" / "rec"
            desktop = (Path(tmp) / ".local" / "share" / "applications" / "com.ryan.rec.desktop").read_text()
            self.assertIn(f"Exec={tmp}/.local/bin/rec\n", desktop)
            self.assertIn("StartupWMClass=com.ryan.rec\n", desktop)
            fake_program(Path(tmp) / ".local" / "bin" / "r2d2", 'echo "$@"\n')
            handed = subprocess.run([str(launcher), "start"], env=env, capture_output=True, text=True, check=True)
            self.assertEqual(handed.stdout, "rec start\n")
            env["REC_INTERNAL"] = "1"
            widget = subprocess.run([str(launcher), "--help"], env=env, capture_output=True, text=True, check=True)
            self.assertIn("rec runs r2d2 rec", widget.stdout)


class home:
    """Point Path.home at a temporary directory."""

    def __init__(self, path: str) -> None:
        self.path = path
        self.previous = None

    def __enter__(self):
        self.previous = os.environ.get("HOME")
        os.environ["HOME"] = self.path

    def __exit__(self, *_exc) -> None:
        if self.previous is None:
            os.environ.pop("HOME", None)
        else:
            os.environ["HOME"] = self.previous


def fake_program(path: Path, body: str) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("#!/bin/sh\n" + body)
    path.chmod(path.stat().st_mode | stat.S_IXUSR)
    return path


def struct_hot() -> bytes:
    return b"\xff\x7f" * 32


if __name__ == "__main__":
    unittest.main()
