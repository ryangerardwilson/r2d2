#!/bin/sh
# Install the rec launcher. The camera program in this directory is the source.
set -eu

here=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
share="${HOME}/.local/share/r2d2"
mkdir -p "${HOME}/.local/bin" "${HOME}/.local/share/applications" "$share"
rm -f "${HOME}/.local/bin/booth" "${HOME}/.local/share/applications/com.ryan.booth.desktop"
cp "${here}/rec.py" "${share}/rec.py"
chmod 0644 "${share}/rec.py"
# rec runs r2d2 rec. r2d2 starts the camera widget with REC_INTERNAL=1.
cat > "${HOME}/.local/bin/rec" <<EOF
#!/bin/sh
if [ "\${REC_INTERNAL:-}" = 1 ]; then
  exec /usr/bin/python3 "${share}/rec.py" "\$@"
fi
if [ -x "\${HOME}/.local/bin/r2d2" ]; then
  exec "\${HOME}/.local/bin/r2d2" rec "\$@"
fi
exec r2d2 rec "\$@"
EOF
chmod 0755 "${HOME}/.local/bin/rec"

cat > "${HOME}/.local/share/applications/com.ryan.rec.desktop" <<EOF
[Desktop Entry]
Name=Rec
Comment=Record the screen with you in the corner
Exec=${HOME}/.local/bin/rec
Icon=camera-web
Terminal=false
Type=Application
Categories=AudioVideo;Recorder;
StartupWMClass=com.ryan.rec
EOF
