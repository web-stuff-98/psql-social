# pSQL-social (my portfolio project for Vue)

# Something went wrong with hosting this. I have no idea how to use docker, i just tried random stuff for a couple of weeks until it worked, now it doesn't, so I will try more random crap for a few weeks until it works again
# https://psql-social.herokuapp.com

## Summary
This is my best filesharing/voip/video chat app. Go-Social-Media is similar except for it uses React and has a public posts feed with nested comments.

You just need a username and password, everything gets deleted automatically twenty minutes after you log out, and you can also manually delete your account. This is an example app for my portfolio.

Also this one is meant to be more similar to Discord, it was going to be a desktop binary using Tauri but I made it a web app instead because nobody wants to run random .exes off the internet. The subscription model for watching for changes is also coded better than Go-Social-Media but there aren't many major differences in the rest of the code.

I coded my own middlewares for authentication and rate limiting because I wanted to make that myself instead of using Fibers middlewares

## Features
 - WebRTC VOIP, screensharing and group video chat
 - Rate limiters for HTTP requests and socket messages (using redis)
 - Refresh tokens stored inside secure httpOnly cookies
 - Filesharing with progress updates
 - Live updates for everything through websocket events using a subscription model and intersection observers
 - Customizable chatrooms
 - Blocks, bans and invitations
 - Validation for everything (inbound + outbound HTTP requests and socket events)
 - Some other features I probably forgot I added because this is a few months old now