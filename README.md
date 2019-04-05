# Project details:

A simple REST API implmentation using golang. Create a post API endpoint which accepts booking date, booking time and party size like below
```sh
 POST /booking/YYYY-MM-DD/HH:MM:SS
   content-type : application-json
   body : { "party_size": Number , "phone": Number , "name" : "String" }
 ```
   
The API should return a boolean success parameter to indicate, whether booking request is a successful or not. For a successful booking, it should return a result object, and in case of failure an error object will be expected. All the responses should have a relevant http status code, like - 200 for success , 4XX and 5XX for errors. The Response schema should be like following
```js
{
  success: Boolean,
  result: {  // optional
    table_id : Number,
    booking_date : "String",
    booking_time : "String",
    no_of_seats: Number,
  }
  errors : { //optional
    reason: String
  }
}