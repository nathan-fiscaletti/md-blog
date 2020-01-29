# md-blog

A simple static-site-generator for bloggers that relies entirely on Markdown

> This project is a WIP project.
> [Go](https://golang.org/doc/install) is required to use this project

[Demo Site](https://blog.nathanf.tk/)

## Setup

1. Clone the repository.
```
$ git clone https://github.com/nathan-fiscaletti/md-blog.git myblog
```
2. Sign up for [Font Awesome](https://fontawesome.com) and configure your `font_awesome_kit` in `config.yml`.
2. Choose a logo and default author image and place them in your `public` directory.
3. Configure your website in `config.yml`. 
4. Write your blog posts in `.md` files and place them in the `posts` directory.
4. Build your website using the following command.
```
./build
```
5. Generated site files will be placed in `./public/`.

## Post Meta Data

At the beginning of each post `.md` file you can place meta data.

It should be written in the format `!!key value`

Example:
```md
!!title My Post Title
!!tags first tag,second tag,third tag
# Place some markdown [here](https://google.com)
```

|Key|Example Value|Required|
|---|---|---|
|`title`|`My Blog Post`|**YES**|
|`tags`|`tag1,tag 2,tag 3`|NO|
|`tags`|`tag1,tag 2,tag 3`|NO|
|`author`|`John Doe`|NO|
|`author_bio`|`I am John Doe`|NO|
|`author_avatar`|`john.jpg`|NO|
