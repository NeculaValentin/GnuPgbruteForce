#!/bin/bash

# This script decrypts a file using a password
# Usage: ./decrypt.sh <file> <password> <start_time>
encrypted_file="$1"
password="$2"
start_time="$3"

# Try to decrypt the file using gpg
echo "$password" | gpg --batch --yes --no-use-agent --passphrase-fd 0 -d "$encrypted_file" >/dev/null 2>/dev/null

# If the exit code is 0, then the password was correct
if [[ $? -eq 0 ]]; then

  current_time=$(date +%s)
  elapsed_seconds=$((current_time - start_time))

  hrs=$((elapsed_seconds / 3600))
  mins=$(( (elapsed_seconds % 3600) / 60))
  secs=$((elapsed_seconds % 60))

  echo "Password found: $password in ${hrs}hrs ${mins}min ${secs}sec"
  pkill -f "decrypt.sh"
fi
