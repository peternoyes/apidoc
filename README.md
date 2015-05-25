# apidoc [![wercker status](https://app.wercker.com/status/0adb8589aef3c6eb84a15be7a3ad08e8/s "wercker status")](https://app.wercker.com/project/bykey/0adb8589aef3c6eb84a15be7a3ad08e8)
apidoc is a simple tool which generate documentation from source code file for REST API, the documentation format is just like http, spaces was ingored.

Currently, this tool only support output markdown format, you should use other tools to convert it to what you needed.

### Install
`go get github.com/cosiner/apidoc`

### Format
All things here:
* @Category {category name}   
   category name for a file, default "global"
* @API {name} @C {category}  
  start a api section, category is optional
* @SubAPI {name}  
   define a subapi, only allow response sections
* @APIIncl {name1, name2, ...}  
   include one or more sub-apis
* @SubResp {name}     
   define a subresponse section
* @RespIncl {name1, name2, ...}    
   include one or more sub-response
* @EndAPI     
   endup a api section, it's mainly used to seperate api documentation from 
   normal code comments, if no code comments, it's not needed  
* @Req  
   define a request section
* @Resp  
   define a response section
* ->  
   identify request/response body

### Usage
The `-e` option specified file extension name,
the '-c' option specified code comments start characters,
with these two options, most language can be supported.

# Examples
Note: Alighment is not a constraint, just for beautifuly.

```Go
// @Category User

// @API create token for authorized user @C Token
//      only for authorized user
// @Req
//     POST /auth/token
//     Authorization:base64(user:password)
// @Resp
//     400 BadRequest
//  -> {"error":"authorization info can't be parsed"}
// @Resp
//     401 Unauthorized
//  -> {"error":"username or password was wrong"}
// @APIIncl generateToken
func CreateAuthToken() {}

// @API create a new account
// @Req
//     POST /account
//   -> {"email":email, "password":password}
// @Resp
//     400 BadRequest
//  -> {"error":"authorized info can't be parsed"}
// @APIIncl generateToken
func CreateAccount() {}

// @SubAPI generateToken
// @RespIncl TokenCreated, ServerError
func GenerateToken() {}

// @SubResp ServerError
//     500 ServerError
// @SubResp TokenCreated
//     201 Created
//  -> {token(string)}
```

# Example Output
```Sh
$ ./apidoc -e md README.md
```
<pre>
### 1. Token
#### 1. create token for authorized user
only for authorized user  
* **Request**
    * POST /auth/token  
      Authorization:base64(user:password)  
* **Response**
    * 400 BadRequest  
      {"error":"authorization info can't be parsed"}  
    * 401 Unauthorized  
      {"error":"username or password was wrong"}  
    * 201 Created  
      {token(string)}  
    * 500 ServerError  

### 2. User
#### 1. create a new account
* **Request**
    * POST /account  
      {"email":email, "password":password}  
* **Response**
    * 400 BadRequest  
      {"error":"authorized info can't be parsed"}  
    * 201 Created  
      {token(string)}  
    * 500 ServerError  

</pre>

