# req
CLI Application for running .http files

[![asciicast](https://asciinema.org/a/623455.png)](https://asciinema.org/a/623455)

## Installation

```shell
go install github.com/go-rq/req
```

## Usage

```shell
# load all requests in the current directory tree
req

# load all requests from the directory tree at the specified path
req ./path/to/dir

req --help
Usage of req:
  -e string
        path to .env file (shorthand)
  -env string
        path to .env file
```

## .http File Syntax

```http request
### <name of request>
< {% <pre-request
javascipt> %} 

<method> <url>
<header>:<value>
...

<request body>

< {% <post-request 
javascript> %}
```

### Scripts

Scripts can be embedded in the `.http` request directly or loaded from

```http
### <name of request>
< path/to/script.js

<method> <url>
<header>:<value>
...

<request body>

< path/to/script.js
```

### Examples

```http request
### Create a User
< {% 
    setEnv('host', 'http://localhost:8000');
    setEnv('name', 'r2d2');
%}
POST {{host}}/users
Content-Type: application/json

{
    "name": "{{name}}"
}

< {% 
    assertTrue("response code is 200", response.status === 200);
%}
```

