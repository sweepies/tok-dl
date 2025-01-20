from json import JSONDecodeError
import json
import shutil
import subprocess
import click
from pathlib import Path
from loguru import logger
import sys
from typing import TextIO
from urllib.parse import urlparse, urlencode
from validators import url as url_validator
import requests
import time
import pause
import dbm

API_BASE_URL = "https://www.tikwm.com/api/"

@click.command()
@click.option("-m", "--metadata-only", is_flag=True, default=False, help="Only download metadata.")
@click.option("-o", "--out", type=click.Path(file_okay=False, writable=True, path_type=Path), default="tiktok", help="Output directory. (default ./tiktok)")
@click.option("-e", "--errors-file", type=click.Path(dir_okay=False, writable=True, path_type=Path), default="errors.log", help="File to save errors to. (default ./errors.log)")
@click.option("--no-cache", is_flag=True, default=False, help="Bypass the bad URL cache.")
@click.argument("input_file", type=click.File())
def main(input_file, metadata_only, out, errors_file, no_cache):
    """TikTok Downloader"""
    
    logger.configure(handlers=[])
    logger.add(sys.stderr, format="{time:HH:mm:ss} <level>{level}</level> <green>{extra}</green> <level>{message}</level>")

    if not metadata_only:
        if not check_wget():
            logger.critical("Could not locate wget binary. Install wget.")
            return 1

    meta_dir = Path(out, "meta")
    if not meta_dir.exists():
        meta_dir.mkdir(parents=True)
        logger.debug(f"Created output directories")
    
    urls = read_input(input_file)
    num_urls = len(urls)
    logger.info(f"Loaded {num_urls} URLs")

    errors = open(errors_file, "w")
    cache = dbm.open(".tiktok_cache", "c")
    cache.keys

    request_time = time.time()
    index = 0
    for url in urls:
        id = url_to_id(url)
        index+= 1
        progress = f"{index}/{num_urls}"

        with logger.contextualize(id=id, progress=progress):

            if not no_cache:
                if check_cache(url, cache):
                    logger.debug("Skipping cached URL")
                    continue

            if test_meta_exists(id, out):
                logger.debug("Metadata exists")
                meta = read_meta(id, out)
            else:
                pause.until(request_time + 1)
                request_time = time.time()

                try:
                    meta = download_meta(id, url, out)
                except JSONDecodeError:
                    logger.error("JSON decode error")
                    write_error_url(url, errors)
                    # cache result
                    write_cache_url(url, cache)
                    continue
                if meta.get("msg") and meta.get("msg").startswith("Free Api Limit"):
                    logger.error("Rate limit reached")
                    write_error_url(url, errors)
                    continue
                elif meta.get("msg") and meta.get("msg").startswith("Url parsing is failed"):
                    logger.error("URL parsing error from API")
                    write_error_url(url, errors)
                    # cache result
                    write_cache_url(url, cache)
                    continue

            if metadata_only:
                logger.info("Download complete")
                continue

            download_url = meta["data"].get("hdplay") or meta["data"].get("play")

            if not download_url:
                logger.error("No download URL")
                write_error_url(url, errors)
                continue

            # we determine whether its a video or gallery by if images are present
            if meta["data"].get("images"):
                download_gallery_post(id, meta, out, logger)
            else:
                download_video_post(id, meta, out, logger)

            logger.info("Download complete")
            
    logger.info("Finished")

def check_wget():
    return shutil.which("wget")

def download_gallery_post(id, meta, out_path: Path, logger):
    
    gallery_dir = Path(out_path, id)
    download_url = meta["data"].get("hdplay") or meta["data"].get("play")

    if not gallery_dir.exists():
        gallery_dir.mkdir()
    
    mp3_path = Path(gallery_dir, f"{id}.mp3")

    if mp3_path.exists():
        logger.debug("Audio exists")
    else:
        logger.debug("Downloading audio")
        subprocess.run(["wget", "--no-verbose", "-O", str(mp3_path), download_url])

    for image_url in meta["data"]["images"]:
        image_name = Path(urlparse(image_url).path).name
        image_path = Path(gallery_dir, image_name)

        if image_path.exists():
            logger.debug("Image exists")
        else:
            logger.debug("Downloading image")
            subprocess.run(["wget", "--no-verbose", "-O", str(image_path), image_url])

def download_video_post(id, meta, out_path: Path, logger):
    download_url = meta["data"].get("hdplay") or meta["data"].get("play")
    mp4_path = Path(out_path, f"{id}.mp4")

    if mp4_path.exists():
        logger.debug("Video exists")
    else:
        logger.debug("Downloading video")
        subprocess.run(["wget", "--no-verbose", "-O", str(mp4_path), download_url])

def write_error_url(url, errors_file):
    errors_file.write(f"{url}\n")
    errors_file.flush()

def write_cache_url(url, cache):
    cache[url] = True

def check_cache(url, cache):
    return url in cache

def url_to_id(url):
    return Path(urlparse(url).path).name

def format_api_url(url):
    q = urlencode({
        "url": url,
        "hd": 1
    })

    return f"{API_BASE_URL}?{q}"

def test_meta_exists(id, out_path: Path):
    return Path(out_path, "meta", f"{id}.json").exists()

def download_meta(id, url, out_path: Path):
    resp = requests.get(format_api_url(url))

    try:
        resp_json = resp.json()

        if resp_json.get("data"):
            with open(Path(out_path, "meta", f"{id}.json"), "w") as meta_file:
                meta_file.write(resp.text)
                meta_file.close()

        return resp_json
    except:
        raise

def read_meta(id, out_path: Path):
    with open(Path(out_path, "meta", f"{id}.json"), "r") as meta_file:
        meta = json.loads(meta_file.read())
        meta_file.close()
        return meta

def read_input(input_file: TextIO):
    urls = []

    while line := input_file.readline():
        line = line.rstrip()
        if line.startswith(("#", "//", "--")):
            continue
        if url_validator(line):
            urls.append(line)
        else:
            logger.debug(f"Skipping invalid URL: {line}")

    urls.sort()
    return urls

if __name__ == '__main__':
    main()