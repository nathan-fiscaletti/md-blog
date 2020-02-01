package main

import(
    "flag"
    "fmt"
    "io/ioutil"
    "regexp"
    "path/filepath"
    "os"
    "strings"
    "errors"
    "net/url"
    "sort"
    "time"

    "github.com/gomarkdown/markdown"
    "github.com/gomarkdown/markdown/parser"
    "github.com/writeas/go-strip-markdown"
    "github.com/grokify/html-strip-tags-go"
    "gopkg.in/yaml.v2"
)

type Config struct {

    Site struct {
        Logo string `yaml:"logo"`
        Title string `yaml:"title"`
        TagLine string `yaml:"tagline"`
        DateFormat string `yaml:"dateformat"`
        SharePlatforms []string `yaml:"share_platforms"`
        FontAwesomeKit string `yaml:"font_awesome_kit"`
        Pagination struct {
            PostsPerPage int `yaml:"posts_per_page"`
        } `yaml:"pagination"`
        SocialURLs []SocialURL `yaml:"social_urls"`
    } `yaml:"site"`

    Author Author `yaml:"author"`

    Templates struct {
        MainTemplate string
        PostTemplate string
        BadgeTemplate string
        PreviewTemplate string
        PreviewFullTemplate string
        SocialURLTemplate string
        ViewBadgeTemplate string
    }

    TagList []string

    // Temporary palceholder for posts while parsing meta data
    // See ParseMetaData function
    Post Post
}

type SocialURL struct {
    Name string `yaml:"name"`
    FABIcon string `yaml:"fab_icon"`
    URL string `yaml:"url"`
}

type Author struct {
    Name string `yaml:"name"`
    Bio string `yaml:"bio"`
    Avatar string `yaml:"avatar"`
}

type Post struct {
    SourceFile string
    Title string
    UrlSafeTitle string
    Tags []string
    Date string
    DateUnix int64
    Content string
    PreviewContent string
    Author Author
}

type PostCollection []Post

func (collection PostCollection) Len() int {
    return len(collection)
}

func (collection PostCollection) Swap(i, j int) {
    collection[i], collection[j] = collection[j], collection[i]
}

func (collection PostCollection) Less(i, j int) bool {
    return collection[i].DateUnix > collection[j].DateUnix
}

func LoadConfig(file string, config *Config) error {
    yamldata,err := ioutil.ReadFile(file)
    if err != nil {
        return err
    }

    err = yaml.Unmarshal(yamldata, config)
    if err != nil {
        return err
    }

    return nil
}

func main() {
    fmt.Printf("]\n")
    fmt.Printf("] Generating Static Site ...\n")
    fmt.Printf("]\n")

    var configFile string
    flag.StringVar(&configFile, "c", "config.yml", "the configuration file")

    fmt.Printf("] Parsing config in " + configFile + "\n")
    var config Config
    err := LoadConfig(configFile, &config)
    if err != nil {
        fmt.Printf("failed to parse config: %v\n", err)
        os.Exit(1)
    }

    Clean("./public/")
    LoadTemplates(&config, "./compiler/templates/")
    posts := ParsePosts(&config, "./posts/", "./public/");
    
    fmt.Printf("] Generating files for all tags\n")
    for _,tag := range config.TagList {
        err := ParseTagPage(&config, posts, tag, "./public/")
        if err != nil {
            fmt.Printf("failed to generate tag page: %v\n", err)
            os.Exit(1)
        }
    }

    err = ParseMainPage(&config, posts, "./public/")
    if err != nil {
        fmt.Printf("failed to generate site index: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("] Done.\n")
}

func Clean(dir string) error {
    fmt.Printf("] Cleaning directory " + dir + "\n")
    d, err := os.Open(dir)
    if err != nil {
        return err
    }

    defer d.Close()

    files, err := d.Readdir(-1)
    if err != nil {
        return err
    }

    for _, file := range files {
        if file.Mode().IsRegular() {
            if filepath.Ext(file.Name()) == ".html" {
                fmt.Printf("]     Deleting " + dir + file.Name() + "\n")
                os.Remove(dir + file.Name())
            }
        }
    }

    return nil
}

func LoadTemplates(config *Config, dir string) {
    post_template,_ := ioutil.ReadFile(dir + "post.tmpl")
    badge_template,_ := ioutil.ReadFile(dir + "badge.tmpl")
    preview_template,_ := ioutil.ReadFile(dir + "post_preview.tmpl")
    preview_full_template,_ := ioutil.ReadFile(dir + "post_preview_full.tmpl")
    main_template,_ := ioutil.ReadFile(dir + "main.tmpl")
    social_url_template,_ := ioutil.ReadFile(dir + "social_url.tmpl")
    view_badge_template,_ := ioutil.ReadFile(dir + "view_tag.tmpl")

    config.Templates.PostTemplate = string(post_template)
    config.Templates.BadgeTemplate = string(badge_template)
    config.Templates.PreviewTemplate = string(preview_template)
    config.Templates.PreviewFullTemplate = string(preview_full_template)
    config.Templates.MainTemplate = string(main_template)
    config.Templates.SocialURLTemplate = string(social_url_template)
    config.Templates.ViewBadgeTemplate = string(view_badge_template)
}

var tags_page_substitution_parsers = map[string]func(*Config, []Post, string) string {
    "PAGE_TITLE": func(config *Config, posts []Post, tag string) string {
        return config.Site.Title + " - " + tag
    },
    "FONT_AWESOME_KIT": func(config *Config, posts []Post, tag string) string {
        return config.Site.FontAwesomeKit
    },
    "COPYRIGHT_YEAR": func(config *Config, posts []Post, tag string) string {
        return fmt.Sprintf("%v", time.Now().Year())
    },
    "TAG": func(config *Config, posts []Post, tag string) string {
        return tag
    },
    "POSTS": func(config *Config, posts []Post, tag string) string {
        data := ""
        for _,post := range posts {
            postIsInTag := false
            for _,postTag := range post.Tags {
                if postTag == tag {
                    postIsInTag = true
                    break
                }
            }

            if postIsInTag {
                post_template := config.Templates.PreviewFullTemplate
                post_template = strings.Replace(post_template, "{{POST_URL}}", post.UrlSafeTitle + ".html", -1)
                post_template = strings.Replace(post_template, "{{POST_TITLE}}", post.Title, -1)
                post_template = strings.Replace(post_template, "{{POST_DATE}}", post.Date, -1)
                post_template = strings.Replace(post_template, "{{POST_PREVIEW_TEXT}}", post.PreviewContent, -1)

                tags := ""
                for _,tag := range post.Tags {
                    template := config.Templates.BadgeTemplate
                    template = strings.Replace(template, "{{BADGE_NAME}}", tag, -1)
                    template = strings.Replace(template, "{{BADGE_URL}}", "__tag__" + url.QueryEscape(strings.ToLower(tag)) + ".html", -1)
                    tags = tags + template + " "
                }
                post_template = strings.Replace(post_template, "{{POST_BADGES}}", tags, -1)
                data = data + post_template
            }
        }

        return data
    },
}

func ParseTagPage(config *Config, posts []Post, tag string, outputdir string) error {
    data := config.Templates.ViewBadgeTemplate

    lines := strings.Split(data, "\n")
    for _,line := range lines {
        pattern := regexp.MustCompile(`({{)(?P<sub>[A-Za-z0-9_-]+)(}})`)
        matches := pattern.FindAllStringSubmatch(line, -1)
     
        for _,match := range matches {
            matchcount := len(match)

            if matchcount > 0 && matchcount < 4 {
                return errors.New(line)
            }

            if matchcount > 0 {
                if _,exists := tags_page_substitution_parsers[match[2]]; exists {
                    parsed := tags_page_substitution_parsers[match[2]](config, posts, tag)
                    data = strings.Replace(data, "{{" + match[2] + "}}", parsed, -1)
                }
            }
        }
    }

    fmt.Printf("]     Generating tag" + outputdir + "__tag__" + url.QueryEscape(strings.ToLower(tag)) + ".html\n")
    ioutil.WriteFile(outputdir + "__tag__" + url.QueryEscape(strings.ToLower(tag)) + ".html", []byte(data), 0755)
    return nil
}

var main_page_substitution_parsers = map[string]func(*Config, []Post) string {
    "SITE_TITLE": func(config *Config, posts []Post) string {
        return config.Site.Title
    },
    "FONT_AWESOME_KIT": func(config *Config, posts []Post) string {
        return config.Site.FontAwesomeKit
    },
    "SITE_LOGO": func(config *Config, posts []Post) string {
        return config.Site.Logo
    },
    "SITE_TAGLINE": func(config *Config, posts []Post) string {
        return config.Site.TagLine
    },
    "COPYRIGHT_YEAR": func(config *Config, posts []Post) string {
        return fmt.Sprintf("%v", time.Now().Year())
    },
    "SOCIAL_URLS": func(config *Config, posts []Post) string {
        data := ""

        for _,socialURL := range config.Site.SocialURLs {
            social_url_template := config.Templates.SocialURLTemplate
            social_url_template = strings.Replace(social_url_template, "{{NAME}}", socialURL.Name, -1)
            social_url_template = strings.Replace(social_url_template, "{{FAB_ICON}}", socialURL.FABIcon, -1)
            social_url_template = strings.Replace(social_url_template, "{{URL}}", socialURL.URL, -1)

            data = data + social_url_template
        }

        return data
    },
    "POSTS": func(config *Config, posts []Post) string {
        data := ""
        for _,post := range posts {
            post_template := config.Templates.PreviewFullTemplate
            post_template = strings.Replace(post_template, "{{POST_URL}}", post.UrlSafeTitle + ".html", -1)
            post_template = strings.Replace(post_template, "{{POST_TITLE}}", post.Title, -1)
            post_template = strings.Replace(post_template, "{{POST_DATE}}", post.Date, -1)
            post_template = strings.Replace(post_template, "{{POST_PREVIEW_TEXT}}", post.PreviewContent, -1)

            tags := ""
            for _,tag := range post.Tags {
                template := config.Templates.BadgeTemplate
                template = strings.Replace(template, "{{BADGE_NAME}}", tag, -1)
                template = strings.Replace(template, "{{BADGE_URL}}", "__tag__" + url.QueryEscape(strings.ToLower(tag))+".html", -1)
                tags = tags + template + " "
            }
            post_template = strings.Replace(post_template, "{{POST_BADGES}}", tags, -1)
            data = data + post_template
        }

        return data
    },
}

func ParseMainPage(config *Config, posts []Post, outputdir string) error {
    data := config.Templates.MainTemplate

    lines := strings.Split(data, "\n")
    for _,line := range lines {
        pattern := regexp.MustCompile(`({{)(?P<sub>[A-Za-z0-9_-]+)(}})`)
        matches := pattern.FindAllStringSubmatch(line, -1)
     
        for _,match := range matches {
            matchcount := len(match)

            if matchcount > 0 && matchcount < 4 {
                return errors.New(line)
            }

            if matchcount > 0 {
                if _,exists := main_page_substitution_parsers[match[2]]; exists {
                    parsed := main_page_substitution_parsers[match[2]](config, posts)
                    data = strings.Replace(data, "{{" + match[2] + "}}", parsed, -1)
                }
            }
        }
    }

    fmt.Printf("] Generating " + outputdir + "index.html\n")
    ioutil.WriteFile(outputdir + "index.html", []byte(data), 0755)
    return nil
}

func ParsePosts(config *Config, dir string, outputdir string) []Post {
    fmt.Printf("] Parsing markdown files in " + dir + "\n")
    var posts []Post
    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if strings.HasSuffix(path, ".md") {
            data,_ := ioutil.ReadFile(path)
            lines := string(data)
            md, err := ParseMetaData(config, lines)
            if err != nil {
                return errors.New(fmt.Sprintf("error while parsing meta data for %v: invalid meta data %v\n", path, err))
            }

            parser := parser.NewWithExtensions(parser.CommonExtensions | parser.Tables | parser.Footnotes | parser.Titleblock | parser.AutoHeadingIDs | parser.SuperSubscript | parser.LaxHTMLBlocks)
            output := markdown.ToHTML([]byte(md), parser, nil)

            preview_content := strip.StripTags(md)
            preview_content = stripmd.Strip(preview_content)
            if len(preview_content) > 425 {
                preview_content = preview_content[0:425] + " ..."
            }

            posts = append(posts, Post{
                SourceFile: path,
                Title: config.Post.Title,
                UrlSafeTitle: config.Post.UrlSafeTitle,
                Tags: config.Post.Tags,
                Date: info.ModTime().Format(config.Site.DateFormat),
                DateUnix: info.ModTime().Unix(),
                Content: string(output),
                PreviewContent: preview_content,
                Author: config.Post.Author,
            })
        }
        return nil
    })

    if err != nil {
        fmt.Printf("error while parsing posts: %v", err)
    }

    // sort posts based on mod time
    sort.Sort(PostCollection(posts))

    for idx,post := range posts {
        var lastPost *Post
        var nextPost *Post
        
        if idx > 0 {
            lastPost = &(posts[idx - 1])
        }

        if idx < len(posts) - 1 {
            nextPost = &(posts[idx + 1])
        }

        html,err := GeneratePostHTML(config, &post, lastPost, nextPost)
        if err != nil {
            fmt.Printf("error while generating post html for %v: %v\n", post.Title, err)
            os.Exit(1)
        }

        fmt.Printf("]     " + post.SourceFile + " -> " + outputdir + post.UrlSafeTitle + ".html\n")
        ioutil.WriteFile(outputdir + post.UrlSafeTitle + ".html", []byte(html), 0755)
    }

    return posts
}

var html_substitution_parsers = map[string]func(*Config, *Post, *Post, *Post)string {
    "PAGE_TITLE": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        return config.Site.Title + " - " + post.Title
    },
    "TITLE": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        return post.Title
    },
    "DATE": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        return post.Date
    },
    "BADGES": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        tags := ""
        for _,tag := range post.Tags {
            template := config.Templates.BadgeTemplate
            template = strings.Replace(template, "{{BADGE_NAME}}", tag, -1)
            template = strings.Replace(template, "{{BADGE_URL}}", "__tag__" + url.QueryEscape(strings.ToLower(tag)) + ".html", -1)
            tags = tags + template + " "
        }
        return tags
    },
    "POST": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        return post.Content
    },
    "AUTHOR_NAME": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        if post.Author.Name != "" {
            return post.Author.Name
        }

        return config.Author.Name
    },
    "AUTHOR_BIO": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        if post.Author.Bio != "" {
            return post.Author.Bio
        }

        return config.Author.Bio
    },
    "AUTHOR_IMAGE": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        if post.Author.Avatar != "" {
            return post.Author.Avatar
        }

        return config.Author.Avatar
    },
    "SOCIAL_SHARE_BUTTONS": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        result := ""
        for _,platform := range config.Site.SharePlatforms {
            result = result + "<a class='a2a_button_" + platform + "'>\n</a>\n"
        }
        return result
    },
    "COPYRIGHT_YEAR": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        return fmt.Sprintf("%v", time.Now().Year())
    },
    "LAST_POST": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        if lastPost != nil {
            tmpl := config.Templates.PreviewTemplate
            tmpl = strings.Replace(tmpl, "{{TITLE}}", lastPost.Title, -1)
            tmpl = strings.Replace(tmpl, "{{DATE}}", lastPost.Date, -1)
            tmpl = strings.Replace(tmpl, "{{POST_PREVIEW}}", lastPost.PreviewContent, -1)
            tmpl = strings.Replace(tmpl, "{{POST_URL}}", lastPost.UrlSafeTitle + ".html", -1)

            return tmpl
        }

        return ""
    },
    "NEXT_POST": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        if nextPost != nil {
            tmpl := config.Templates.PreviewTemplate
            tmpl = strings.Replace(tmpl, "{{TITLE}}", nextPost.Title, -1)
            tmpl = strings.Replace(tmpl, "{{DATE}}", nextPost.Date, -1)
            tmpl = strings.Replace(tmpl, "{{POST_PREVIEW}}", nextPost.PreviewContent, -1)
            tmpl = strings.Replace(tmpl, "{{POST_URL}}", nextPost.UrlSafeTitle + ".html", -1)

            return tmpl
        }

        return ""
    },
    "FONT_AWESOME_KIT": func(config *Config, post *Post, lastPost *Post, nextPost *Post) string {
        return config.Site.FontAwesomeKit
    },
}

func GeneratePostHTML(config *Config, post *Post, lastPost *Post, nextPost *Post) (string,error) {
    // substitution regex: \{\{[a-zA-Z0-0_-]+\}\}
    data := config.Templates.PostTemplate
    lines := strings.Split(data, "\n")
    for _,line := range lines {
        pattern := regexp.MustCompile(`({{)(?P<sub>[A-Za-z0-9_-]+)(}})`)
        matches := pattern.FindAllStringSubmatch(line, -1)
        
        for _,match := range matches {
            matchcount := len(match)

            if matchcount > 0 && matchcount < 4 {
                return "",errors.New(line)
            }

            if matchcount > 0 {
                if _,exists := html_substitution_parsers[match[2]]; exists {
                    parsed := html_substitution_parsers[match[2]](config, post, lastPost, nextPost)
                    data = strings.Replace(data, "{{" + match[2] + "}}", parsed, -1)
                }
            }
        }
    }

    return data,nil
}

var meta_data_parsers = map[string]func(*Config, string) {
    "title": func(config *Config, value string) {
        config.Post.Title = value
        config.Post.UrlSafeTitle = url.QueryEscape(strings.ToLower(value))
    },
    "author": func(config *Config, value string) {
        config.Post.Author.Name = value
    },
    "author_bio": func(config *Config, value string) {
        config.Post.Author.Bio = value
    },
    "author_avatar": func(config *Config, value string) {
        config.Post.Author.Avatar = value
    },
    "tags": func(config *Config, value string) {
        config.Post.Tags = strings.Split(value, ",")
        for _,tag := range config.Post.Tags {
            exists := false
            for _,ex_tag := range config.TagList {
                if ex_tag == tag {
                    exists = true
                    break
                }
            }

            if !exists {
                config.TagList = append(config.TagList, tag)
            }
        }
    },
}

func ParseMetaData(config *Config, post string) (string,error) {
    lines := strings.Split(post, "\n")

    config.Post = Post{}

    for _,line := range lines {
        pattern := regexp.MustCompile(`!!(?P<key>title|author|author_bio|author_avatar|tags)\s(?P<val>.*)`)
        matches := pattern.FindAllStringSubmatch(line, -1)

        for _,match := range matches {
            matchcount := len(match)

            if matchcount > 0 && matchcount < 3 {
                return "",errors.New(line)
            }

            if matchcount > 0 {
                if _,exists := meta_data_parsers[match[1]]; exists {
                    meta_data_parsers[match[1]](config, match[2])
                }

                post = strings.Replace(post, line, "", 1)
            }
        }
    }

    return post,nil
}

