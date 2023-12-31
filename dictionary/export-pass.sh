#!/bin/bash

output_file="my_passwords.txt"


# Create or overwrite the output file
> output_file
max_length=4  # Change this to the maximum length you desire

# Iterate over each file
for file in files/*.txt; do
    # Process the text and append to the output file
    echo $file
    cat $file | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z]//g' | grep -v '^$' >> "temp_passwords.txt"

done
# Prefix each line by its length, sort by this prefix, then strip the prefix
awk -v max="$max_length" '{if (length <= max) print length, $0}' temp_passwords.txt | sort -n | cut -d' ' -f2- | uniq > "$output_file"

rm temp_passwords.txt  # Remove the temporary file