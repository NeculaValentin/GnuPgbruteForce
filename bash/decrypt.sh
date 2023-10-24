#!/bin/bash

encrypted_file="$1"
password="$2"
start_time="$3"

echo "$password" | gpg --batch --yes --passphrase-fd 0 -d "$encrypted_file" >/dev/null 2>/dev/null

if [[ $? -eq 0 ]]; then

  current_time=$(date +%s)
  elapsed_seconds=$((current_time - start_time))

  hrs=$((elapsed_seconds / 3600))
  mins=$(( (elapsed_seconds % 3600) / 60))
  secs=$((elapsed_seconds % 60))

  echo "Password found: $password in ${hrs}hrs ${mins}min ${secs}sec"
  pkill -f "decrypt.sh"
fi
