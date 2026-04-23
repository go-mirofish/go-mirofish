#!/bin/bash

# Security Check Script
# Scans for sensitive files that should not be committed

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BRIGHT='\033[1m'
NC='\033[0m'

# Sensitive file patterns
SENSITIVE_PATTERNS=(
    "^\.env$"
    "^\.env\.local$"
    "^\.env\.[^.]*\.local$"
    "^\.env\.production$"
    "^\.env\.development$"
    "^\.env\.test$"
    "^\.env\.staging$"
    "\.env$"
    "\.secrets?$"
    "secrets/"
    "\.secret$"
    "\.envrc$"
    "\.env\.backup$"
    "\.env\.[^.]*\.backup$"
    "\.pem$"
    "\.key$"
    "\.cert$"
    "\.p12$"
    "\.pfx$"
    "\.jks$"
    "\.keystore$"
    "credentials\.json$"
    "service-account\.json$"
    "auth\.json$"
    "token\.json$"
    "\.gpg$"
    "\.pgp$"
    "\.cursor/mcp\.json$"
)

# Allowed example files
ALLOWED_EXAMPLES=(
    "\.env\.example$"
    "\.env\.template$"
    "\.env\.sample$"
    "\.env\.example\.local$"
    "\.cursor/mcp\.json\.example$"
    "\.cursor/mcp\.example\.json$"
)

print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

is_sensitive_file() {
    local file="$1"
    
    # Check if it's an allowed example file first
    for pattern in "${ALLOWED_EXAMPLES[@]}"; do
        if [[ "$file" =~ $pattern ]]; then
            return 1  # Not sensitive
        fi
    done
    
    # Check against sensitive patterns
    for pattern in "${SENSITIVE_PATTERNS[@]}"; do
        if [[ "$file" =~ $pattern ]]; then
            return 0  # Sensitive
        fi
    done
    
    return 1  # Not sensitive
}

check_staged_files() {
    local sensitive_files=()
    
    while IFS= read -r file; do
        if [[ -n "$file" ]] && is_sensitive_file "$file"; then
            sensitive_files+=("$file")
        fi
    done < <(git diff --cached --name-only 2>/dev/null)
    
    echo "${sensitive_files[@]}"
}

check_working_directory() {
    local sensitive_files=()
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local file="${line:3}"
            if is_sensitive_file "$file"; then
                local status="${line:0:2}"
                sensitive_files+=("$file|$status")
            fi
        fi
    done < <(git status --porcelain 2>/dev/null)
    
    printf '%s\n' "${sensitive_files[@]}"
}

check_gitignore() {
    if [[ ! -f ".gitignore" ]]; then
        echo "missing"
        return
    fi
    
    if grep -qE "^\.env$|^\.env[[:space:]]" .gitignore 2>/dev/null; then
        echo "ok"
    else
        echo "missing_env"
    fi
}

# Main execution
print_color $BRIGHT "🔒 Security Check for Git Repository"
print_color $BRIGHT "===================================="

# Check if in git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_color $RED "❌ Not in a git repository!"
    exit 1
fi

# Check .gitignore
print_color $BLUE ""
print_color $BLUE "📋 Checking .gitignore..."
gitignore_status=$(check_gitignore)
case "$gitignore_status" in
    "missing")
        print_color $YELLOW "⚠️  .gitignore file not found!"
        ;;
    "missing_env")
        print_color $YELLOW "⚠️  .gitignore may not properly exclude .env files"
        print_color $YELLOW "   Ensure .gitignore contains: .env"
        ;;
    "ok")
        print_color $GREEN "✅ .gitignore properly configured for .env files"
        ;;
esac

# Check staged files
print_color $BLUE ""
print_color $BLUE "🔍 Checking staged files..."
staged_sensitive=($(check_staged_files))

if [[ ${#staged_sensitive[@]} -gt 0 ]]; then
    print_color $RED "❌ Found ${#staged_sensitive[@]} sensitive file(s) in staging area:"
    for file in "${staged_sensitive[@]}"; do
        print_color $RED "   - $file"
    done
    print_color $YELLOW ""
    print_color $YELLOW "⚠️  ACTION REQUIRED: Remove these files from staging:"
    print_color $YELLOW "   git reset HEAD <file>"
    exit 1
else
    print_color $GREEN "✅ No sensitive files in staging area"
fi

# Check working directory
print_color $BLUE ""
print_color $BLUE "🔍 Checking working directory..."
working_sensitive=($(check_working_directory))

if [[ ${#working_sensitive[@]} -gt 0 ]]; then
    print_color $YELLOW "⚠️  Found ${#working_sensitive[@]} sensitive file(s) in working directory:"
    for file_info in "${working_sensitive[@]}"; do
        local file=$(echo "$file_info" | cut -d'|' -f1)
        local status=$(echo "$file_info" | cut -d'|' -f2)
        print_color $YELLOW "   - $file ($status)"
    done
    print_color $CYAN ""
    print_color $CYAN "💡 These files will be blocked if you try to commit them"
    print_color $CYAN "   Ensure they are in .gitignore"
fi

# Summary
print_color $BRIGHT ""
print_color $BRIGHT "📊 Security Check Summary:"
total_sensitive=$((${#staged_sensitive[@]} + ${#working_sensitive[@]}))
if [[ $total_sensitive -eq 0 ]]; then
    print_color $GREEN "✅ Security check passed!"
    exit 0
else
    print_color $RED "❌ Security check failed!"
    exit 1
fi

