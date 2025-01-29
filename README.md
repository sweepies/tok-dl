<h1 align="center">
  Tok-DL: A TikTok Downloader that actually works
</h1>
<p align="center">
  <img src="https://github.com/user-attachments/assets/9d57a8a9-18d2-4751-b573-466f57607840" />
</p>

## Features
- Saves both video and gallery posts, including all of their metadata
- Highest quality, no watermarks
- Caches already-downloaded and unavailable posts to avoid fetching them again

## Installation
via [mise](https://mise.jdx.dev) (recommended)
```shell
mise use -g go:github.com/sweepies/tok-dl
```

## Usage
Tok-DL takes input in the form of newline-separated links. This format is the same as is contained in TikTok personal data downloads. It will ignore commented lines.

```shell
NAME:
   tok-dl - A TikTok Downloader that actually works

USAGE:
   tok-dl [global options] INPUT_FILE

GLOBAL OPTIONS:
   --metadata-only, -m        only download metadata (default: false)
   --out-dir value, -o value  output directory (default: "./tiktok")
   --no-cache                 bypass the cache; don't skip alreadty actioned urls (default: false)
   --debug                    show debug logs (default: false)
   --help, -h                 show help                Show this message and exit.
```

## Limitations
- Since Tok-DL utilizes the [TiKWM](https://www.tikwm.com/) API, there is a limit of 5,000 requests per day, and 1 per second. Tok-DL automatically handles this on a second-by-second basis, but the program will stop if you hit the daily limit. Thankfully, you can easily pick up where you left off.
