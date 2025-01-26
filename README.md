<h1 align="center">
  Tok-DL: A TikTok Downloader that actually works
</h1>
<p align="center">
  <img src="https://github.com/user-attachments/assets/9d57a8a9-18d2-4751-b573-466f57607840" />
</p>
<div align="center">

[![PyPI version](https://badge.fury.io/py/tok-dl.svg?icon=si%3Apython)](https://badge.fury.io/py/tok-dl)

</div>

## Features
- Saves both video and gallery posts, including all of their metadata
- Highest quality, no watermarks
- Caches already-downloaded and unavailable posts to avoid fetching them again

## Installation
```shell
pip install tok-dl
```

## Usage
Tok-DL takes input in the form of newline-separated links. This format is the same as is contained in TikTok personal data downloads. It will ignore commented lines.

```shell
Usage: tok-dl [OPTIONS] INPUT_FILE

  TikTok Archiver

Options:
  -m, --metadata-only     Only download metadata.
  -o, --out DIRECTORY     Output directory. (default ./tiktok)
  -e, --errors-file FILE  File to save errors to. (default ./errors.log)
  --no-cache              Bypass the bad URL cache.
  --help                  Show this message and exit.
```

## Limitations
- Since Tok-DL utilizes the [TiKWM](https://www.tikwm.com/) API, there is a limit of 5,000 requests per day, and 1 per second. Tok-DL automatically handles this on a second-by-second basis, but you will begin seeing errors if you hit the daily limit. Thankfully, you can easily pick up where you left off by feeding `errors.log` back in to Tok-DL as an input file.
