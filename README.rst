CA
==

CA is a local cache system for "curlable" sites such as cheat.sh.

CA uses curl to download information per usual, but then caches the result for future access. This reduces server load and also makes pages accessible when offline.


How to install
--------------

Build from source code::

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

    # use Curl user agent, and update every 30 mins
    $ alias news="ca -age=30 -A ''  http://getnews.tech"

    # get our IP but don't cache. Some sites check for curl user-agent
    $ alias me="ca -f -A 'curl/7.88.1' ifconfig.me"

    # and our city...
    $ alias city="ca ifconfig.co/city"

    # guess what this one does...
    $ alias weather="ca -age=120 wttr.in/Berlin"
