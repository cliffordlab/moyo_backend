
## Table of Contents
- [1. API Documentation](#3-api-documentation)
  - [1.1. Users](#31-salesforce)
    - [Moyo User Registration](#moyo-user-registration)
    - [User Login](#user-login)
  + [1.2. AWS](#32-aws)
    - [Upload to S3](#upload-to-s3)
- [2. Contributors](#4-contributors)

# 1. API Documentation
This is the API documentation for the back end of Amoss App.
The data will be stored in AWS S3 bucket.

## 1.1. Users

### Moyo User Registration

*This route is present for the registration of Moyo users*

**Path:**

Request Type | URL | Content-Type
--- | --- | ---
  POST |  http://localhost:4200/api/moyo/register | multipart/form-data

  POST |  http://localhost:4200/api/createAdmin
  POST |  http://localhost:4200/api/createCoordinator
  POST |  http://localhost:4200/api/createPatient

**Params:**

Name | Type | Description
--- | --- | ---
Authorization | string | **Required.** Authorization token to create a new user.
participantID | long | **Required.** User's registered ID.
password | string | **Required.** Password provided must be at least 6 characters long.


**Status Codes:**

Code | Type | Description
---|---|---
200 | Success | Server has processed the request and has successfully updated the user.
401 | Error | Unauthorized. Incorrect username and/or password combination.

**Example Body:**

```
Header |
--- | 
Authorization : Bearer ey...s
```
Body |
--- | 
{
  "participantID":yourParticipantID,
  "password":"yourPassword"
}

**Example Response:**

```
{"success":"patient participant created"}
{"success": "you have completed upload to S3Bucket}
```

**Example Failure Response:**

```
{
    "error": "json parsing error",
    "error description": "key or value of json is formatted incorrectly"
}
```

### User Login

*This route is present for the login of users*

**Path:**

Request Type | URL
--- | ---
POST |  http://localhost:4200/loginParticipant

**Params:**

Name | Type | Description
--- | --- | ---
participantID | long | **Required.** User's registered ID. Log in with participant ID.
password | string | **Required.** Password provided must be at least 6 characters long.

**Status Codes:**

Code | Type | Description
---|---|---
200 | Success | Server has processed the request and has successfully updated the user.
401 | Error | Unauthorized. Incorrect username and/or password combination.

**Example Body:**

```
{
  "participantID":yourParticipantID,
  "password":"yourPassword"
}
```
**Example Response:**

```
{
    "token": "eyJ...........co",
    "capacity": "coordinator",
    "participantID": "yourParticipantID",
    "study": "study_name",
    "isConsented": "false"
}
```

**Example Failure Response:**

```
{
    "error": "json parsing error",
    "error description": "key or value of json is formatted incorrectly"
}
```

## 1.2. AWS

### Upload to S3

*This route is present for the Amazon S3 file uploads*

**Path:**

Request Type | URL
--- | ---
POST | http://localhost:4200/upload_s3

**Headers:**

Name | Type | Description
--- | --- | ---
Authorization | string | **Required.** Mars token.
weekMillis | long | **Not Required.** Timestamp. Some studies doesn't require an weekMillis

**Params:**

Name | Type | Description
--- | --- | ---
path | string | **Required.** Only the path where the file will be uploaded. <b>S3 bucket name is not needed<\b>. 
                              Authorization token provides the S3 bucket information and name.
upload | file | **Required.** Files to be uploaded.

**Status Codes:**

Code | Type | Description
---|---|---
200 | Success | Server has processed the request and has successfully updated the user.
422 | Error | Unprocessable Entry. Specified parameters are invalid.

**Example Header:**

```
Authorization: Mars fd..Y,
```

**Example Body:**

```
{
  "path": "S3PathFolder",
  "weekMillis": 534118400000
  "upload": "YourFile"
}
```

**Example Response:**

```
{
  "success": "you have completed upload to awsS3Bucket"
}
```

# 2. Contributors

Daniel Phan && Tony Nguyen

Update: August 15, 2021

**README** documentation by **_Daniel Phan_**
