bundlescript
============

`bundlescript` bundles all the JavaScript files referenced in an HTML file to a single
JavaScript file, removes all script tags and adds a script tag at the end of the HTML
file that references the bundled JavaScript file.

Usage
-----

	bundlescript --htdocs /path/to/your/htdocs --htmlin index.html --htmlout index-new.html --jsout /js/bundle.js

This will read the file `/path/to/your/htdocs/index.html`, bundle all its JavaScript sources
to a single file, and write to resulting HTML and JavaScript files to `/path/to/your/htdocs/index-new.html`
resp. `/path/to/your/htdocs/js/bundle.js`.

The HTML input file and output file can be the same.

License
-------

See the file LICENSE for license information.


Author
------

Andreas Krennmair <ak@synflood.at>
