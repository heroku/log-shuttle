#!/bin/sh

tempdir=$(mktemp -d)
control_dir="$tempdir/DEBIAN"
mkdir -p -m 0755 "$control_dir"
cat > "$control_dir/control" <<EOF
Package: log-shuttle
Version: 0.0.1
Architecture: amd64
Maintainer: "Ryan R. Smith" <ryan@heroku.com>
Section: heroku
Priority: optional
Description: Move logs from the Dyno to the Logplex.
EOF

install_path="$tempdir/usr/local/bin"
mkdir -p "$install_path"

install log-shuttle $install_path/log-shuttle
fakeroot dpkg-deb --build "$tempdir" .
