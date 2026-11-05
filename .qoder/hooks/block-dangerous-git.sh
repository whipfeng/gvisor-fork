#!/bin/bash
# Block dangerous git operations that can lose work
# No jq dependency - uses grep to parse JSON

INPUT=$(cat)

# Extract command from JSON (simple grep approach)
COMMAND=$(echo "$INPUT" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"command"[[:space:]]*:[[:space:]]*"//' | sed 's/"$//')

# Block git reset --hard (can lose uncommitted work)
if echo "$COMMAND" | grep -qE 'git[[:space:]]+reset[[:space:]]+--hard'; then
    echo "Blocked: 'git reset --hard' can lose committed and uncommitted work." >&2
    echo "Use 'git reset --soft' or 'git reset --mixed' instead." >&2
    echo "Or confirm with user first before executing." >&2
    exit 2
fi

# Block git clean -fd (can delete untracked files)
if echo "$COMMAND" | grep -qE 'git[[:space:]]+clean[[:space:]].*-f'; then
    echo "Blocked: 'git clean' can delete untracked files permanently." >&2
    echo "Confirm with user first." >&2
    exit 2
fi

# Block git checkout -- (can discard changes)
if echo "$COMMAND" | grep -qE 'git[[:space:]]+checkout[[:space:]]+--'; then
    echo "Blocked: 'git checkout --' discards uncommitted changes." >&2
    echo "Use 'git stash' instead or confirm with user first." >&2
    exit 2
fi

# Allow other commands
exit 0
