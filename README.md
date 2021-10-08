# websitechanges

Alerts you via email about a website change, with an image of the change.

After writing a [website change script in Python](https://github.com/schollz/websitechanges), I realized I could make it much better using Go+Node. This code was very helpful for me to get registered for a high-demand [pottery class](https://schollz.com/blog/pottery/).

In this version I'm using [puppeteer](https://github.com/puppeteer/puppeteer) to do the capturing. The puppeteer code first filters out any ads (to prevent new things appearing in screen) and then captures an image of a CSS-selected content. The change-detection happens in Go and sends an email with an image of the change when any is detected.

## Example

Here is an example of a change detected from the New York times headlines.

<p align="center">
<img src="https://user-images.githubusercontent.com/6550035/89665256-3d615a00-d88d-11ea-8fd3-dda51a4cd3f5.jpg">
</p>

The previous state is shown in red and the new state is shown in green.

The `config.json` used was (note the folder key is optional and is used to customize the folder name of images for easier recognition.):

```javascript
{
    "watchers": [
    {
        "url": "https://www.nytimes.com",
        "css": "h2 > span",
        "folder": "nytimes"
    }]
}
```

## Usage

First make sure you have `node` and `go` installed on your system.

Then download and build:

```
> git clone https://github.com/schollz/websitechanges
> cd websitechanges
> go build -v
```

Now copy the config file and fill it in (you don't have to include email, just erase that section if you don't need it):

```
> cp config-example.json config.json
> vim config.json
```

Now run. The first time you run it will download a HOSTS file that is used for filtering adds and it will install puppeteer if not installed.

```
> ./websitechanges
```

By default this will run every 5 minutes. You can adjust the delay by passing the `-delay 10` where 10 is the number of minutes to wait before the next run.

```
> ./websitechanges -delay 10
```

It automatically generates diff images everytime it encounters a change.

## Contributing

Pull requests are welcome. Feel free to...

- Revise documentation
- Add new features
- Fix bugs
- Suggest improvements

## License

MIT
