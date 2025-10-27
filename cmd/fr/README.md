# fr
The `fr` utility 3 optional arguments and walks the directory you run it from, examines every regular file that `file --mime-type` reports as a text type, prints each matching line, and optionally replaces the pattern.

If the command line ends with `-f` (i.e. `fr FROM TO -f`) the program writes the replacements.

If `-f` is absent (i.e. `fr FROM TO`) the program does not modify any file – it only prints every line that contains a match, colouring the matched fragment in red and the file name in yellow.

## Search/Replace Types

1. Single-argument search: `fr 'foo.*bar'`

Prints all matching lines with matches highlighted in red. Note that the user can supply any valid regex.

2. Show-only mode: `fr 'FROM' 'TO'`

Highlight occurrences without writing changes.

3. Replace-and-write mode: `fr 'FROM' 'TO' -f`

Replace occurrences in all text files.

