# maven-client
## Description
This CLI reads a list of maven group artifact version (GAV) coordinates and returns an ordered list of first order and transitive GAV coordinates within the runtime and compile scopes.

example

maven-client -config config.txt -input input.txt
## Command line flags
- config
- input

Both flags expect a path to a respective file
## Config
- repourl
- username
- password
## Input
A file containing new line separated entries to query against.

The format is group:artifact:version
## Output
A file containing new line separated alphabetically sorted GAV coordinates
