#!/usr/bin/env bash
# Rename this template for your project: Maven coordinates and Java package.
#
#   ./rename.sh <groupId> <artifactId>
#   ./rename.sh com.acme acme-auth      # package becomes com.acme.acmeauth
#
# Run it once, right after you copy the template. After a successful rename it
# deletes itself, so it never lands in your project's git history.
# Class names (AuthkitApplication, AuthkitProperties) and the `authkit.*`
# property prefix stay as they are; rename them in your IDE if you want to.
set -euo pipefail
SELF=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")
cd "$(dirname "$SELF")"

if [ $# -ne 2 ]; then
  echo "usage: $0 <groupId> <artifactId>   e.g. $0 com.acme acme-auth" >&2
  exit 2
fi
NEW_GROUP=$1
NEW_ARTIFACT=$2
if ! [[ $NEW_GROUP =~ ^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$ ]]; then
  echo "groupId must be lower-case dot-separated words, e.g. com.acme" >&2
  exit 2
fi
if ! [[ $NEW_ARTIFACT =~ ^[a-z][a-z0-9]*(-[a-z0-9]+)*$ ]]; then
  echo "artifactId must be lower-case words joined by '-', e.g. acme-auth" >&2
  exit 2
fi

# What the tree uses now, read from the tree so the script can run on any copy.
OLD_PKG=$(sed -n 's/^package \(.*\);$/\1/p' "$(find src/main/java -name AuthkitApplication.java)")
OLD_GROUP=$(perl -0ne 'print $1 if m{</parent>\s*<groupId>([^<]+)</groupId>}' pom.xml)
OLD_ARTIFACT=$(perl -0ne 'print $1 if m{</parent>\s*<groupId>[^<]+</groupId>\s*<artifactId>([^<]+)</artifactId>}' pom.xml)
if [ -z "$OLD_PKG" ] || [ -z "$OLD_GROUP" ] || [ -z "$OLD_ARTIFACT" ]; then
  echo "could not read the current package or coordinates" >&2
  exit 1
fi

NEW_PKG="$NEW_GROUP.${NEW_ARTIFACT//-/}"
if [ "$OLD_PKG" = "$NEW_PKG" ] && [ "$OLD_GROUP" = "$NEW_GROUP" ] && [ "$OLD_ARTIFACT" = "$NEW_ARTIFACT" ]; then
  echo "already named $NEW_GROUP:$NEW_ARTIFACT"
  exit 0
fi
OLD_PATH=${OLD_PKG//.//}
NEW_PATH=${NEW_PKG//.//}

# Replace a literal string in the files of the given list.
replace() { # <old> <new> <file>...
  local old=$1 new=$2
  shift 2
  OLD=$old NEW=$new perl -pi -e 's/\Q$ENV{OLD}\E/$ENV{NEW}/g' "$@"
}

# 1. Package declarations, imports and the paths README links to.
FILES=$(grep -rlF -e "$OLD_PKG" -e "$OLD_PATH" src README.md pom.xml Dockerfile 2>/dev/null || true)
if [ -n "$FILES" ]; then
  # shellcheck disable=SC2086
  replace "$OLD_PKG" "$NEW_PKG" $FILES
  # shellcheck disable=SC2086
  replace "$OLD_PATH" "$NEW_PATH" $FILES
fi

# 2. Maven coordinates (the project's, not the Spring Boot parent's).
OLD_GROUP=$OLD_GROUP OLD_ARTIFACT=$OLD_ARTIFACT NEW_GROUP=$NEW_GROUP NEW_ARTIFACT=$NEW_ARTIFACT \
  perl -0pi -e 's{(</parent>\s*<groupId>)\Q$ENV{OLD_GROUP}\E(</groupId>\s*<artifactId>)\Q$ENV{OLD_ARTIFACT}\E(</artifactId>)}{$1$ENV{NEW_GROUP}$2$ENV{NEW_ARTIFACT}$3}' pom.xml

# 3. The artifact name where it appears: jar name, app name, docs.
replace "$OLD_ARTIFACT" "$NEW_ARTIFACT" pom.xml Dockerfile Makefile docker-compose.yml README.md src/main/resources/application.yml

# 4. Move the source trees. Through a temporary directory, because the new
#    package may sit inside the old one.
for base in src/main/java src/test/java; do
  [ -d "$base/$OLD_PATH" ] || continue
  mv "$base/$OLD_PATH" "$base/.rename-tmp"
  (cd "$base" && rmdir -p "$(dirname "$OLD_PATH")" 2>/dev/null || true)
  mkdir -p "$base/$NEW_PATH"
  mv "$base/.rename-tmp"/* "$base/$NEW_PATH"/
  rmdir "$base/.rename-tmp"
done

rm -rf target
echo "renamed $OLD_GROUP:$OLD_ARTIFACT ($OLD_PKG) -> $NEW_GROUP:$NEW_ARTIFACT ($NEW_PKG)"
rm -f -- "$SELF"
echo "removed rename.sh; next: ./mvnw test"
