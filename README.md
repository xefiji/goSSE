# Context

goSSE is a tiny SSE server written in Go during covid crisis.

It serves four main objectives:

- Expose a light server to handle Server sent events in my personal and professional apps
- Know more about the language: concurrency, networking, routing, middlewares
- Be a playground for people (friends and co workers) to participate
- Not getting bored while you're containment at home

# SSE

Mozilla doc says it all: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events


## What are they ?

Unidirectional messages objects sent from the server to the connected clients, received through the EventSource API.

## When to use them ?

When you need to send updates/notifs to browsers from server, without having to maintain a bidirectional connexion (which you would do with websockets).

# Example

![Demo](/sse_demo.gif)

# Setup

Clone the project:

`git clone https://github.com/xefiji/goSSE.git`

Fill env vars:

`cp .env.dist .env && nano .env`

```
# App's vars
SSE_USERNAME= # there will be the user allowed to access the server
SSE_PASSWORD= # there will be the user's password
SSE_APP_KEY= # secret key to crypt JWT token
SSE_PORT=80 # Port where to expose the API
ALLOWED_ORIGIN= # For CORS. Wildcard won't work (cookie setup)

# Reverse proxy's vars
COMPOSE_PROJECT_NAME= # well known
SSE_URL= # well known
WHITELIST= # well known
TRUSTED_PROXIES= # well known
```

Run:

`docker-compose up --build`

# Consume

Use https://github.com/xefiji/tinyClientSSE if needed. Provides a light and humble way to connect to the server and set a callback for each event.


```js
    //instantiate
    var client = new SSEClient("https://sse.serveradress.tld","sse","my_user","my_password", "my_user_id");
    
    //set callback method that will handle every received events
    client.setCallback(function(event){
        console.log(event); //example
    })
    
    //run (will log in and run EventStream)
    client.run();
```

# Send

## Log in first:


Request would be
```
POST /login HTTP/1.1
Host: sse.serveradress.tld
Content-Type: application/json
{
	"username": "my_user",
	"password": "my_password"
}
```

Php implementation:

```php
$curl = curl_init();

curl_setopt_array($curl, array(
  CURLOPT_URL => "https://sse.serveradress.tld/login",
  CURLOPT_CUSTOMREQUEST => "POST",
  CURLOPT_POSTFIELDS =>"{\n\t\"username\": \"my_user\",\n\t\"password\": \"my_password",
  CURLOPT_HTTPHEADER => array(
    "Content-Type: application/json",    
  ),
));

$response = curl_exec($curl);
curl_close($curl);
echo $response;//token is here
```

## Now that you have the token

Request would be

```
POST /broadcast HTTP/1.1
Host: sse.serveradress.tld
Content-Type: application/json
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InNzZV91c2VyIiwiZXhwIjoxNTg2MjcwNjUxfQ.1rJWDePXHjSPMsFOgjTolrr-isQ_HVM9anZiABwXzmY
Content-Type: application/json
{
	"content": "Awesome test message to broadcast to all connected browsers",
	"type": "sse", # optional: feel free to broadcast to only one type of event
	"user": "a99d4b4e-e5c4-42b4-888d-651862c599f6" # optional: feel free to broadcast to only one user
}
```

Php implementation

```php
<?php

$curl = curl_init();

curl_setopt_array($curl, array(
  CURLOPT_URL => "https://sse.serveradress.tld/broadcast",
  CURLOPT_CUSTOMREQUEST => "POST",
  CURLOPT_POSTFIELDS =>"{\n\t\"content\": \"Awesome test message to broadcast to all connected browsers\",\n\t\"type\": \"sse\",\n\t\"user\": \"a99d4b4e-e5c4-42b4-888d-651862c599f6\"\n}",
  CURLOPT_HTTPHEADER => array(
    "Content-Type: application/json",
    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InNzZV91c2VyIiwiZXhwIjoxNTg2MjcwNjUxfQ.1rJWDePXHjSPMsFOgjTolrr-isQ_HVM9anZiABwXzmY",
    "Content-Type: application/json",
  ),
));

$response = curl_exec($curl);

curl_close($curl);
echo $response;
```

Returns 201, 401, 400

# Todos

- Https of course
- Better user auth handling
- Keep track of all users notifs in a mongo bdd
- else ?