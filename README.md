# apidoc
apidoc is a simple tool which generate api documentation from source file,
the documentation format is just like http.

Currently, this tool only support output markdown format, you should use other tools to convert it to what you needed.

### Install
`go get github.com/cosiner/apidoc`

### Format
All things here:
* @API {description}             
  start a api section
* @SubResp {name}     
   define a subresponse section
* @RespIncl {name}  
   include a subresponse
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
// @API create token for authorized user
// @Req
//     POST /auth/token
//     Authorization:base64(user:password)
// @Resp
//     400 BadRequest
//  -> {"error":"authorization info can't be parsed"}
// @Resp
//     401 Unauthorized
//  -> {"error":"username or password was wrong"}
// @RespIncl generateToken
func CreateAuthToken() {}

// @API create a new account
// @Req
//     POST /account
//   -> {"email":email, "password":password}
// @Resp
//     400 BadRequest
//  -> {"error":"authorized info can't be parsed"}
// @RespIncl generateToken
func CreateAccount() {}

// @SubResp generateToken
// @Resp
//     201 Created
//  -> {"token":"1234567890"}
// @Resp
//     500 ServerError
func GenerateToken() {}
```

# Example Output
```Sh
$ ./apidoc -e md README.md
```
<pre>
### 1. create token for authorized user
* **Request**
    * POST /auth/token  
      Authorization:base64(user:password)  
* **Response**
    * 400 BadRequest  
      {"error":"authorized info can't be parsed"}  
    * 401 Unauthorized  
      {"error":"username or password was wrong"}  
    * 201 Created  
      {"token":"1234567890"}  
    * 500 ServerError  

### 2. create a new account
* **Request**
    * POST /account  
      {"email":email, "password":password}  
* **Response**
    * 400 BadRequest  
      {"error":"authorized info can't be parsed"}  
    * 201 Created  
      {"token":"1234567890"}  
    * 500 ServerError
</pre>

