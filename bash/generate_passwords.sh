#!/bin/bash

generate_passwords() {
    local max_length=4  # Adjust this as needed

    for (( length=1; length<=$max_length; length++ )); do
        case $length in
            1) echo {a..z} ;;
            2) echo {a..z}{a..z} ;;
            3) echo {a..z}{a..z}{a..z} ;;
            4) echo {a..z}{a..z}{a..z}{a..z} ;;
        esac
    done | tr ' ' '\n'
}

file_path="passwords.txt"
if [[ -f "$file_path" ]]; then
    echo "File $file_path exists. Replacing it..."
    rm -f "$file_path"
fi

generate_passwords > "$file_path"
