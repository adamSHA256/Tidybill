#!/usr/bin/env bash
# Bump TidyBill version, update changelog, commit, tag, push.
#
# Usage:
#   scripts/release.sh                    patch bump (default)
#   scripts/release.sh --bump minor
#   scripts/release.sh --bump major
#   scripts/release.sh --version 0.6.0    explicit version
#   scripts/release.sh --dry-run          show what would change, no writes
#   scripts/release.sh --no-push          commit + tag locally, don't push
#   scripts/release.sh --skip-check       skip `make check`
#   scripts/release.sh --no-prompt        skip Claude/LLM prompt step
#
# Preflight: must be on main, clean tree, in sync with origin/main, target tag
# unused. Then bumps every version string. For the changelog: writes a self-
# contained prompt to /tmp/tidybill-changelog-prompt.md (commits since last
# tag + style reference from prior entries) — paste into a fresh Claude/LLM
# instance, which writes the result to /tmp/tidybill-changelog.md. The script
# waits for ENTER then opens that file in $EDITOR for a review pass (or falls
# back to a commit-list template if you skipped the LLM step). Runs
# `make check`, asks for confirmation, then commits + tags + pushes (main
# first, then tag — so CI builds from a commit that's already on origin).

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BUMP=patch
EXPLICIT_VERSION=""
DRY_RUN=0
NO_PUSH=0
SKIP_CHECK=0
NO_PROMPT=0

PROMPT_FILE=/tmp/tidybill-changelog-prompt.md
RESULT_FILE=/tmp/tidybill-changelog.md

while [ $# -gt 0 ]; do
  case "$1" in
    --bump)        BUMP="$2"; shift 2 ;;
    --bump=*)      BUMP="${1#*=}"; shift ;;
    --version)     EXPLICIT_VERSION="$2"; shift 2 ;;
    --version=*)   EXPLICIT_VERSION="${1#*=}"; shift ;;
    --dry-run)     DRY_RUN=1; shift ;;
    --no-push)     NO_PUSH=1; shift ;;
    --skip-check)  SKIP_CHECK=1; shift ;;
    --no-prompt)   NO_PROMPT=1; shift ;;
    -h|--help)     sed -n '3,22p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)             echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

say()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!\033[0m  %s\n' "$*"; }
die()  { printf '\033[1;31mERR\033[0m %s\n' "$*" >&2; exit 1; }
ask()  { local p="$1" a; printf '%s ' "$p" >&2; read -r a </dev/tty; printf '%s' "$a"; }

# ---------------------------------------------------------------- preflight ---

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
[ "$BRANCH" = "main" ] || die "must be on 'main', currently on '$BRANCH'"

git diff --quiet || die "uncommitted changes — commit or stash first"
git diff --cached --quiet || die "staged changes — commit or reset first"
[ -z "$(git ls-files --others --exclude-standard)" ] || \
  die "untracked files present — clean up first (git status)"

say "fetching origin"
git fetch --tags --quiet origin
LOCAL=$(git rev-parse @)
REMOTE=$(git rev-parse @{u})
[ "$LOCAL" = "$REMOTE" ] || die "local main diverges from origin/main — pull/rebase first"

# --------------------------------------------------------------- versioning ---

CUR="$(perl -ne 'print $1 if /const Version = "([^"]+)"/' internal/config/config.go)"
[ -n "$CUR" ] || die "could not parse current version from internal/config/config.go"
say "current version: $CUR"

if [ -n "$EXPLICIT_VERSION" ]; then
  NEW="${EXPLICIT_VERSION#v}"
else
  IFS=. read -r MAJ MIN PAT <<<"$CUR"
  case "$BUMP" in
    major) MAJ=$((MAJ+1)); MIN=0; PAT=0 ;;
    minor) MIN=$((MIN+1)); PAT=0 ;;
    patch) PAT=$((PAT+1)) ;;
    *)     die "unknown --bump '$BUMP' (major|minor|patch)" ;;
  esac
  NEW="$MAJ.$MIN.$PAT"
fi
[[ "$NEW" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "invalid new version '$NEW'"
say "new version:     $NEW"

TAG="v$NEW"

if git rev-parse "$TAG" >/dev/null 2>&1; then
  die "local tag $TAG already exists. Delete first: git tag -d $TAG"
fi
if git ls-remote --tags origin "$TAG" 2>/dev/null | grep -q "refs/tags/$TAG$"; then
  die "remote tag $TAG already exists. Delete first:
       gh release delete $TAG --yes --cleanup-tag
   or: git push --delete origin $TAG"
fi

# ----------------------------------------------------------------- bumping ---

# Replace exactly $CUR with $NEW in $file at the position matching $marker.
# $marker is a perl regex containing literal \Q...\E text. First match only.
bump() {
  local file="$1" marker="$2"
  if [ ! -f "$file" ]; then die "missing file: $file"; fi
  if ! grep -qP "$marker" "$file"; then
    die "pattern not found in $file (expected: $marker)"
  fi
  if [ "$DRY_RUN" = "1" ]; then
    echo "  would update: $file"
  else
    perl -i -pe "BEGIN{\$d=0} if(!\$d && /$marker/){s/\\Q$CUR\\E/$NEW/; \$d=1}" "$file"
    echo "  updated: $file"
  fi
}

say "bumping files"
bump internal/config/config.go         'const Version = "\Q'"$CUR"'\E"'
bump desktop/src-tauri/tauri.conf.json '"version": "\Q'"$CUR"'\E"'
bump desktop/package.json              '"version": "\Q'"$CUR"'\E"'
bump desktop/src-tauri/Cargo.toml      '^version = "\Q'"$CUR"'\E"'
bump README.md                         'TIDYBILL v\Q'"$CUR"'\E'

# Cargo.lock — only the tidybill-desktop entry; leave unrelated crates alone.
if [ "$DRY_RUN" = "1" ]; then
  echo "  would update: desktop/src-tauri/Cargo.lock (tidybill-desktop entry)"
else
  perl -i -0pe 's/(name = "tidybill-desktop"\nversion = ")\Q'"$CUR"'\E(")/${1}'"$NEW"'${2}/' \
    desktop/src-tauri/Cargo.lock
  echo "  updated: desktop/src-tauri/Cargo.lock"
fi

# ---------------------------------------------------------------- changelog ---

LAST_TAG="v$CUR"
COMMITS="$(git log --no-merges --pretty='- %s' "$LAST_TAG"..HEAD 2>/dev/null || true)"

# Build a self-contained prompt for a fresh LLM session that knows nothing
# about this repo. Includes commits since last tag + the most recent two
# CHANGELOG entries as a style reference.
write_changelog_prompt() {
  local out="$1" style
  style="$(awk '/^## v/ { c++; if (c > 2) exit } c >= 1 { print }' CHANGELOG.md)"
  cat >"$out" <<EOF
You are writing a CHANGELOG.md entry for TidyBill $TAG.

TidyBill is a desktop/mobile invoicing app for the Czech market. Users are
small business owners; UI strings are in Czech.

# Commits since $LAST_TAG

$COMMITS

# Style reference — most recent CHANGELOG.md entries (match this voice)

$style

# Rules

- Group bullets under: \`### Fixed\` (bug fixes), \`### New\` (features),
  \`### Build\` (CI / build / tooling — usually omit unless visibly relevant
  to users).
- Combine commits that describe the same change into one bullet.
- Write user-visible language. Don't quote a commit subject verbatim if it's
  terse or scoped (e.g. \`wizard:\`, \`invoice(list):\`) — rewrite for a
  non-developer reader.
- Czech UI labels stay in Czech, in quotes, the way prior entries do.
- Skip purely internal commits with no user impact (refactors, lint, tests,
  most dep bumps). Skip \`release:\` and \`Merge pull request\` commits.
- 1–2 sentences per bullet. No marketing fluff.

# Output

Write the result to \`$RESULT_FILE\`. Content of that file must be exactly
the markdown that should be prepended to CHANGELOG.md:

  ## $TAG

  ### Fixed
  - ...

  ### New
  - ...

No preamble, no trailing notes — just the markdown.
EOF
}

if [ "$DRY_RUN" = "1" ]; then
  say "would open editor for CHANGELOG.md entry"
  [ -n "$COMMITS" ] && { echo "  prefill candidates:"; echo "$COMMITS" | sed 's/^/    /'; }
  if [ "$NO_PROMPT" != "1" ]; then
    echo "  would also write LLM prompt to: $PROMPT_FILE"
  fi
else
  TPL="$(mktemp -t tidybill-changelog.XXXXXX.md)"
  trap 'rm -f "$TPL"' EXIT

  # Clear any stale Claude output from a previous run.
  rm -f "$RESULT_FILE"

  if [ "$NO_PROMPT" != "1" ]; then
    write_changelog_prompt "$PROMPT_FILE"
    say "wrote LLM prompt to $PROMPT_FILE"
    cat <<MSG

  To draft the changelog with Claude (or any LLM):
    1) cat $PROMPT_FILE   # or open it; copy the full content
    2) paste into a new chat (no prior context needed)
    3) the model writes the result to $RESULT_FILE
    4) come back here and press ENTER — I'll open it for review

  Or just press ENTER now to write the changelog by hand instead.
MSG
    printf '  > '
    read -r _ </dev/tty
  fi

  if [ -f "$RESULT_FILE" ] && [ -s "$RESULT_FILE" ]; then
    say "found $RESULT_FILE — using as initial draft"
    cp "$RESULT_FILE" "$TPL"
  else
    {
      echo "## $TAG"
      echo
      if [ -n "$COMMITS" ]; then
        echo "<!-- commits since $LAST_TAG (delete this line and edit below) -->"
        echo "### Fixed"
        echo "$COMMITS"
      else
        echo "### Fixed"
        echo "- "
      fi
      echo
      echo "<!-- optional sections — delete if unused"
      echo "### New"
      echo "- "
      echo
      echo "### Build"
      echo "- "
      echo "-->"
    } >"$TPL"
  fi

  EDITOR="${EDITOR:-vi}"
  say "opening \$EDITOR for $TAG changelog (save & quit when done)"
  "$EDITOR" "$TPL"

  # strip HTML comments
  perl -i -0pe 's/<!--.*?-->//gs' "$TPL"
  # collapse 3+ blank lines to 1
  perl -i -0pe 's/\n{3,}/\n\n/g' "$TPL"

  if ! grep -qP '^- \S' "$TPL"; then
    warn "changelog entry has no bullets."
    if [ "$(ask 'continue with empty changelog? [y/N]')" != "y" ]; then
      die "aborted by user (no version files were committed)"
    fi
  fi

  # Prepend new entry to CHANGELOG.md, preserving the leading "# Changelog" line.
  awk -v tpl="$TPL" '
    NR==1 { print; print ""; while ((getline l < tpl) > 0) print l; print ""; next }
    NR==2 && /^[[:space:]]*$/ { next }   # drop the original blank after header
    { print }
  ' CHANGELOG.md > CHANGELOG.md.new
  mv CHANGELOG.md.new CHANGELOG.md
  echo "  updated: CHANGELOG.md"
fi

# -------------------------------------------------------------------- check ---

if [ "$DRY_RUN" != "1" ] && [ "$SKIP_CHECK" != "1" ]; then
  say "running 'make check' (skip with --skip-check)"
  if ! make check; then
    warn "make check failed."
    echo "Files are bumped but not committed. Fix issues, then re-run."
    echo "Or revert: git restore ."
    exit 1
  fi
fi

# ----------------------------------------------------- diff & confirmation ---

say "diff to be committed:"
git --no-pager diff --stat
echo

if [ "$DRY_RUN" = "1" ]; then
  say "dry run complete — no files were changed (changelog editor was skipped)"
  exit 0
fi

action="commit + tag $TAG"
[ "$NO_PUSH" = "1" ] || action="$action + push"
if [ "$(ask "$action ? [y/N]")" != "y" ]; then
  warn "aborted before commit. Files remain bumped on disk."
  echo "Revert with: git restore ."
  exit 1
fi

# ------------------------------------------------ commit, tag, push (last) ---

git add -A
git commit -m "release: $TAG"
git tag -a "$TAG" -m "Release $TAG"

if [ "$NO_PUSH" = "1" ]; then
  say "committed + tagged locally. Push when ready:"
  echo "    git push origin main && git push origin $TAG"
  exit 0
fi

say "pushing main"
git push origin main
say "pushing tag $TAG (this kicks off the release workflow)"
git push origin "$TAG"

say "done. Watch the build:"
echo "    gh run watch"
echo "    gh release view $TAG --web"
