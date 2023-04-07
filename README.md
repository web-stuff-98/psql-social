# pSQL-social (my portfolio project)

https://psql-social.herokuapp.com

## Features:

- Everything is live
- Voip/Video chat
- Screen sharing
- Invitations
- Friend requests
- Filesharing
- Blocks/Bans
- Profile pictures
- Room pictures

## Technical features:

- Cascaded deletes
- Prepared SQL statements
- Typescript types for socket events
- Validation for everything
- HTTP & Socket rate limiters (from scratch)
- Chunked file uploads
- Download streams

### It's go-vue-chat but 99% rewritten and a lot faster since I used postgreSQL instead of MongoDB. Also I used fasthttp and fasthttp/websocket instead of Gorilla. Originally I was using plain fasthttp but I changed to fiber because I was having problems serving the static files, but then I had the same problem with fiber, so I just ended up hosting the frontend on a seperate server.

## If you cannot find a room made by another user, that is because they have to invite you to it. And I just realized I added the private feature for no reason since you have to be a member of a room to find it anyway.

### The client folder structure could be improved a bit, by moving some functions into stores. Improvements could be made to the socket server, maybe by using Sync Map instead of mutex locks, also there is a client performance improvement that could be made, by JSON parsing the socket message once instead of parsing it on every listener. I didn't write any tests because I wanted to finish this project as fast as possible, it took around 3 weeks.
