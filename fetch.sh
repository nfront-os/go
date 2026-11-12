#!/bin/bash
# Download Go's release (sources.txt) to dl/, check its signature (keys/<key>.pub) and
# sha256.txt, unpack it to go/ and make that a repository of its own: the release as
# the commit tagged with its version (go/VERSION), the port on the branch plan9 from
# patches/go (git am).  scripts/patches.sh writes the patches back after a commit there.
# go/ is left as it is if it exists.
# usage: ./fetch.sh
set -e
cd "$(dirname "$0")"
mkdir -p dl/sig
while IFS=$'\t' read -r name url key; do
	case $name in ''|\#*) continue ;; esac
	f=dl/$(basename "$url")
	[ -s "$f" ] || curl -fL -o "$f" "$url"
	sig=dl/sig/$(basename "$url").asc
	[ -s "$sig" ] || curl -fL -o "$sig" "$url.asc"
	gpg --dearmor < "keys/$key.pub" > "dl/sig/$key.gpg"
	gpgv --keyring "$PWD/dl/sig/$key.gpg" "$sig" "$f" 2>/dev/null ||
		{ echo "fetch: bad signature on $f" >&2; exit 1; }
	grep " $f\$" sha256.txt | sha256sum -c --quiet ||
		{ echo "fetch: $f: not the sha256 of sha256.txt" >&2; exit 1; }
	[ -d "$name" ] && { echo "fetch: $name/ is there, left as it is"; continue; }
	mkdir "$name.tmp"
	tar -xf "$f" -C "$name.tmp" --strip-components=1
	ver=$(head -1 "$name.tmp/VERSION")
	(cd "$name.tmp" && git init -q -b release && git add -A &&
	 git -c user.name=release -c user.email=release@localhost commit -qm "$ver (the release archive)" &&
	 git tag "$ver" && git switch -q -c plan9 && git am -q ../patches/$name/*.patch)
	mv "$name.tmp" "$name"
	echo "fetch: $name/: $ver, branch plan9 with $(ls patches/$name/*.patch | wc -l) patches"
done < sources.txt
