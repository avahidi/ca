CA
==

CA is a local cache system for "curlable" sites such as cheat.sh.

When CA downloads the information, it caches the result for future access. This reduces server load and also makes pages accessible when offline.


How to install
--------------

Build from source code::

    sudo apt install golang
    go install github.com/avahidi/ca@latest


How to use it
-------------

Simply replace "curl" with "ca"::

    $ curl cht.sh/rust/pointers
    Use references when you can, use pointers when you must. If you're not...

    $ ca cht.sh/rust/pointers
    Use references when you can, use pointers when you must. If you're not...

Now the page is cached and next time you access it, it will be retrieved from local cache instead of network.

More interestingly, you can create helper functions or aliases for different tasks. Here is an example for Golang (using bash)::

    $ alias go?="ca --prefix https://cht.sh/go/"
    $ go? strings
    ...
    $ go? :learn
    ...

Here are some other aliases you can try::

    # get news, update every 30 mins
    $ alias news="ca -age=30  http://getnews.tech"

    # get our IP but don't cache. Set the user-agent to some really old curl
    $ alias me="ca -f -A 'curl/1.0.0' ifconfig.me"

    # In case you wake up and have no idea where you are...
    $ alias city="ca ifconfig.co/city"

    # guess what this one does...
    $ alias weather="ca -age=120 wttr.in/Berlin"

    # I don't own any crypto, but in case you do...
    $ alias broke="ca -age=60 rate.sx/ETH"

    # whois light?
    $ alias where?="ca -f -prefix=ipinfo.io/"
    $ where? 127.0.0.1
    $ where? 8.8.4.4

    # QR codes:
    $ alias qrcode="ca -age=99999 --prefix=qrenco.de/"
    $ qrcode "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

For more examples see https://github.com/chubin/awesome-console-services

Templates
---------

To get you started, CA comes with a few template queries::

    $ ca @weather
    $ ca @weather dublin
    $ ca @go slices
    $ ca @btc

To see all available templates::

   $ ca @help

To add your own, see the configuration file (~/.config/ca.conf).
