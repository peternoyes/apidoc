# apidoc [![wercker status](https://app.wercker.com/status/0adb8589aef3c6eb84a15be7a3ad08e8/s "wercker status")](https://app.wercker.com/project/bykey/0adb8589aef3c6eb84a15be7a3ad08e8)
apidoc is a simple tool which generate documentation from source code file for REST API, the documentation format is just like http, spaces was ingored.

If sub-{api, response, header} not found, error message will be printed to Stderr with light red color.

Currently, this tool only support output markdown format, you should use other tools to convert it to what you needed.

### Install
`go get github.com/cosiner/apidoc`

### Format
All things here:
* `@Category` {category name}   
   category name for a file, default "global"
* `@API` {name} @C {category}  
  start a api section, category is optional
* `@SubAPI` {name}  
   define a subapi, only allow response sections
* `@APIIncl` {name1, name2, ...}    
   include one or more sub-apis
* `@Header` {name}  
    define common headers
* `@HeaderIncl` {name1, name2, ...}  
    include one or more headers
* `@SubResp` {name}     
   define a subresponse section
* `@RespIncl` {name1, name2, ...}    
   include one or more sub-response
* `@EndAPI`     
   endup a api section, it's mainly used to seperate api documentation from 
   normal code comments, if no code comments, it's not needed  
* `@Req`  
   define a request section
* `@Resp`  
   define a response section
* `->``  
   identify request/response body

### Usage
The `-e` option specified file extension name,
the `-c` option specified code comments start characters.
With these two options, almost support all languages.

The `-f` option specified file to save, default output to standard output.

# Examples
Note: Alighment is not a constraint, just for beautifuly.

```
// @Category User

// @API create token for authorized user @C Token
//      only for authorized user
// @Req
//     POST /auth/token
//     @HeaderIncl BasicAuth
// @RespIncl UnAuth
// @APIIncl generateToken
func CreateAuthToken() {}

// @API create a new account
// @Req
//     POST /account
//   ->{
//        email(string), 
//        password(string)
//     }
// @RespIncl UnAuth
// @APIIncl generateToken
func CreateAccount() {}

// @Header BasicAuth
//     Authorization:base64(user(string):password(string))

// @SubAPI generateToken
// @RespIncl TokenCreated, ServerError
func GenerateToken() {}

// @SubResp ServerError
//     500 ServerError
// @SubResp UnAuth
//     401 Unauthorized
//   ->{
//        error:"invalid access token"
//     }
// @SubResp TokenCreated
//     201 Created
//   ->{
//        token(string)
//     }
```

# Example Output
```Sh
$ ./apidoc -e md README.md
```


#### 1. User
##### 1. create a new account
* **Request**
    * POST /account  
      {email(string), password(string)}  
* **Response**
    * 401 Unauthorized  
      {error:"invalid access token"}  
    * 201 Created  
      {token(string)}  
    * 500 ServerError  

#### 2. Token
##### 1. create token for authorized user
only for authorized user  
* **Request**
    * POST /auth/token  
      Authorization:base64(user(string):password(string))  
* **Response**
    * 401 Unauthorized  
      {error:"invalid access token"}  
    * 201 Created  
      {token(string)}  
    * 500 ServerError  

