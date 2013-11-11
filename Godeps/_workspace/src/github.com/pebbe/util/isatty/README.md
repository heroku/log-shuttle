Test program for IsTerminal() in package github.com/pebbe/util

This is how it should work (on linux): 

    ~ go get github.com/pebbe/util 
    ~ go get github.com/pebbe/util/isatty 
    ~ isatty 
    stdin:  true 
    stdout: true 
    stderr: true 
    ~ isatty | cat 
    stdin:  true 
    stdout: false 
    stderr: true 
    ~ isatty 2> /dev/null 
    stdin:  true 
    stdout: true 
    stderr: false 
    ~ isatty < ~/.bashrc 
    stdin:  false 
    stdout: true 
    stderr: true 
    ~ echo | isatty 
    stdin:  false 
    stdout: true 
    stderr: true
