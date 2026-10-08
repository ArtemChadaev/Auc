var token = response.body.data.access_token
client.global.set("auth_token", token)