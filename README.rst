CA
==

CA is a local cache for "curlable" sites such as cheat.sh, wttr.in, and many others.

When CA downloads a page, it is cached for future access. This reduces server load, speeds up subsequent requests, and makes pages accessible even when you are offline.
       

Usage
-----

Simply replace "curl" with "ca":

.. code-block:: console

    $ curl cht.sh/rust/pointers
    Use references when you can, use pointers when you must. If you're not...

    $ ca cht.sh/rust/pointers
    Use references when you can, use pointers when you must. If you're not...

Now the page is cached and next time you access it, it will be retrieved from local cache instead of network.

Adding parameters
~~~~~~~~~~~~~~~~~

The real power of *ca* is unlocked with shell aliases that accept parameters. To use these you first split an URL into a blueprint plus one or more parameters:

.. code-block:: bash

    $ ca "cht.sh/rust/<item>" pointers
    Use references when you can, use pointers when you must. If you're not...

This by itself is not very useful, but when you put this into a helper function or an alias things get more interesting:

.. code-block:: bash
    
    $ alias rust?="ca https://cht.sh/rust/\<item\>"
    $ rust? strings
    ...
    $ rust? lifetime
    ...

Note that depending on the environment, you might need to escape < and >.

More examples
~~~~~~~~~~~~~

Here are some other aliases you can try:

.. code-block:: bash

    # get news, update every 30 mins
    $ alias news="ca -age=30  http://getnews.tech"

    # get our IP but don't cache it, and set user-agent to some really old curl
    $ alias me="ca -f -A 'curl/1.0.0' ifconfig.me"

    # In case you wake up and have no idea where you are...
    $ alias city="ca ifconfig.co/city"

    # guess what this one does...
    $ alias weather="ca -age=120 wttr.in/Berlin"

    # I don't own any crypto, but in case you do...
    $ alias broke="ca -age=60 rate.sx/ETH"

    # whois-ish :)
    $ alias where?="ca -f ipinfo.io/\<ip\>"
    $ where? 127.0.0.1
    $ where? 8.8.4.4

    # QR codes:
    $ alias qrcode="ca -age=99999 qrenco.de/\<item\>"
    $ qrcode "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

For more examples see https://github.com/chubin/awesome-console-services

Templates
---------

*ca* comes with a few built-in templates to get you started.

.. code-block:: console

    $ ca @weather
    $ ca @weather dublin
    $ ca @go slices
    $ ca @btc
    $ ca @help # this will list all available templates

To add your own, see the configuration file (~/.config/ca.conf).


How to install
--------------

Build from source code:

.. code-block:: console

    sudo apt install golang
    go install github.com/avahidi/ca@latest

This will install *ca* to you ~/go/bin/

License
-------

This project is licensed under the GNU General Public License version 2. See the `LICENSE <LICENSE>`_ file for details.

