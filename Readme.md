The program scans github repo for a aws secret leaks.

Repository owner and name are hardcoded in the main function

records about scanned commits are written to the DB to avoid scan on rerun

currently program doesn't do anything special to optimize run after abortion


Next step to improve: create table to store info about branches - remember the oldest commit that was processed

