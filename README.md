
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

## 1.1. Users

### Moyo User Registration

*This route is present for the registration of Moyo users*

**Path:**

Request Type | URL | Content-Type
--- | --- | ---
 POST |  https://amoss.emory.edu/dev/api/moyo/register | multipart/form-data
  POST |  https://amoss.emory.edu/prod/api/moyo/register | multipart/form-data


**Params:**

Name | Type | Description
--- | --- | ---
email | string | **Required.** User's email address.
phone | string | **Required.** User's phone number.
password | string | **Required.** Password provided must be at least 6 characters long.
upload | string | **Required.** CSV file of consent form and demographic questionnaire.


**Status Codes:**

Code | Type | Description
---|---|---
200 | Success | Server has processed the request and has successfully updated the user.
401 | Error | Unauthorized. Incorrect username and/or password combination.

**Example Body:**

```
email:tester5@email.com
phone:7707777777
password:tester
upload:consent.csv
```

**Example Response:**

```
{"success":"patient participant created"}
{"success": "you have completed upload to amoss_mhealth"}
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
POST |  https://amoss.emory.edu/dev/loginParticipant
POST |  https://amoss.emory.edu/prod/loginParticipant


**Params:**

Name | Type | Description
--- | --- | ---
participantID | long | User's registered ID. Login with participant ID or email.
email | string | User's email address. Login with participant ID or email. 
password | string | **Required.** Password provided must be at least 6 characters long.

**Status Codes:**

Code | Type | Description
---|---|---
200 | Success | Server has processed the request and has successfully updated the user.
401 | Error | Unauthorized. Incorrect username and/or password combination.

**Example Body:**

```
{
  "participantID": "9977",
  "email":" ",
  "password": "tester"
}
```

OR

```
{
  "participantID": "0",
  "email":"tonynguyen.dev1@gmail.com",
  "password": "tester"
}
```

OR

```
{
  "participantID": "123456789",
  "password": "tester"
}
```
**Example Response:**

```
{
    "token":"eyJhbGciOiJIUzI1NiIsIc23kpXVCJ9.eyJwYXJ0aWNpcGFudF9pZCI6OTk4ODAwMDAwMCwiY2FwYWNpdHkiOiJjb29yZGluYXRvciIsInN0dWR5IjoidGVzdCIsImV4cCI6MTU2NTg5MTgyMiwiaXNzIjoibG9jYWxob3N0OjgwODAifQ.pMJppHKjUPOp0qF4ErldbHkzjOI8gaG9MEZ-oj_UHyU", 
    "capacity":"coordinator",
    "participantID":"7777777"
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
POST | https://amoss.emory.edu/upload

**Headers:**

Name | Type | Description
--- | --- | ---
Authorization | string | **Required.** Mars token.
weekMillis | long | **Required.** Timestamp

**Params:**

Name | Type | Description
--- | --- | ---
upload | file | **Required.** Files to be uploaded.

**Status Codes:**

Code | Type | Description
---|---|---
200 | Success | Server has processed the request and has successfully updated the user.
422 | Error | Unprocessable Entry. Specified parameters are invalid.

**Example Header:**

```
Authorization: Mars fdsfsdafeyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpZAI6OX0.1YMgT2O8ccKdqvrJph1AcSPeLJpRlVvEgITTXxKWrZY,
weekMillis: 534118400000
```

**Example Body:**

```
{
  "upload": "534118400000.mz"
}
```

**Example Response:**

```
{
  "success": "you have completed upload to amoss_mhealth"
}
```

# 2. Contributors

Christoper Wainwright Aaron && Tony Nguyen

**README** documentation by **_Tony Nguyen_**
