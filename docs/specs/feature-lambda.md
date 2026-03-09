# Requirements Specification

## feature: lambda

## requirement

* create a simple cli tool, call it `rosactl`
* support command, `rosacli lambda create hello` 
* create lambda hello will: 
  1. using the current aws context
  2. check if there is a lambda function named `hello` already exists
  3. if not exist, create an aws lambda function `hello`
  4. the lambda function when invoked, it should output `hello world`
  5. the lambda function is implemented with python
* support command: `rosactl lambda invoke hello`
  1. it should be able to fund the hello lambda function in the current aws context
  2. it should invoke that lambda function
* support command: `rosactl lambda delete hello`
  1. this command should search for the lambda function `hello`
  2. delete the lambda function
* support command: `rosactl lambda list`
  1. list the current lambda functions by name and date
