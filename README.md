# sandstorm-znc

Port of ZNC to [Sandstorm][1]. This is alpha quality, but I'm dogfooding
it and it basically works.

## License

Apache-2.0; see the `LICENSE` file.

# Design Notes

IRC isn't a web based protocol, so building a Sandstorm app does IRC is
slightly more complicated that most Sandstorm apps. We still want
leverage sandstorm for authentication and authorization. We do this by
listening on a websocket instead of a raw TCP port, and have users use
[websocket-proxy][2] to connect.

# Using

To use sandstorm-znc, you must be an administrator for your sandstorm
installation. This is because IRC Idler requires raw network access,
which only an administrator can grant. Additionally, connecting to it
is a little weird; see the Design Notes section above.

Each irc network you want to connect to must run in its own grain. To
set up a new network:

* Create a new ZNC grain
* Fill out the settings for the IRC server on the grain's web interface.
  For example, to connect to freenode, you would supply:
  * Host: irc.freenode.net
  * Port: 6667 for unencrypted, 6697 for TLS
  * Check the TLS box or not, depending on whether you want to use it
    (recommended).
* Click on the "Request Network Access" button, and grant network access
  in the dialog that sandstorm presents
* You will be presented with a websocket URL you can use to connect. You
  can get a traditional IRC client to connect to this by using
  [websocket-proxy][2]:

      websocket-proxy -listen :6000 -url ${websocket_url}

...and then pointing your IRC client at localhost port 6000.

[1]: https://sandstorm.io
[2]: https://github.com/zenhack/websocket-proxy
