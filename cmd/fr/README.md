# fr
The `fr` utility 3 optional arguments and walks the directory you run it from, examines every regular file that `file --mime-type` reports as a text type, prints each matching line, and optionally replaces the pattern.

If the command line ends with `-f` (i.e. `fr FROM TO -f`) the program writes the replacements.

If `-f` is absent (i.e. `fr FROM TO`) the program does not modify any file – it only prints every line that contains a match, colouring the matched fragment in red and the file name in yellow.

