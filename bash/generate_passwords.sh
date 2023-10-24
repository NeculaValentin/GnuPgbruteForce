#!/bin/bash

# This script generates a file with all possible passwords
generate_passwords() {
    local max_length=4  # Adjust this as needed

    # This is a bit ugly, but it works, and it's fast
    for (( length=1; length<=$max_length; length++ )); do
        case $length in
            1) echo {a..z} ;;
            2) echo {a..z}{a..z} ;;
            3) echo {a..z}{a..z}{a..z} ;;
            4) echo {a..z}{a..z}{a..z}{a..z} ;;
        esac
    done | tr ' ' '\n'
}

# This is the file that will contain all possible passwords
file_path="passwords.txt"
if [[ -f "$file_path" ]]; then
    echo "File $file_path exists. Replacing it..."
    rm -f "$file_path"
fi

# Generate the passwords and save them to the file
generate_passwords > "$file_path"
