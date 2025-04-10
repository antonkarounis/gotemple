# gotemple
A helpful manager for go templates



Slightly opinionated and constrained.

Set up two directories, in different locations for the following:

1. a normal directory structure of page templates, as you would do in a web directory like `/var/www/mysite/`
2. a flat directory of included templates

so for example take the following layout:

/www/index.html
    /about.html


/include/header.html
        /footer.html
        /menu.html


Both index.html and about.html can both share header, footer, and menu.




If a "base.html" file is found in the includes, all page templates will be reconfigured to be executed through it. This encapsulates the complexity of handling `block` statements in templates.

It leverages *govalidtemple* to validate that the datamodels passed into the template `Execute()` method is the expected shape to properly generate the page. 



