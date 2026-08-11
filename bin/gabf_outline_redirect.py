#!/usr/bin/env python3
"""Redirect requests to the newest Grace Ambassadors outline PDF."""

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from re import findall
from urllib.error import URLError
from urllib.request import Request, urlopen


LISTING_URL = "https://graceambassadors.com/audio/"
LISTEN_ADDRESS = "127.0.0.1"
LISTEN_PORT = 8091
USER_AGENT = "prsmusa-gabf-outline/1.0"


def latest_outline_url() -> str:
    request = Request(LISTING_URL, headers={"User-Agent": USER_AGENT})
    with urlopen(request, timeout=15) as response:
        page = response.read().decode("utf-8", errors="replace")

    # The listing is newest-first. Ignore placeholder links such as audio/.pdf.
    urls = findall(r"href=['\"]([^'\"]+\.pdf)['\"]", page, flags=0)
    for url in urls:
        if "/audio/.pdf" not in url:
            return url
    raise ValueError("No valid outline PDF found on the Grace Ambassadors audio index")


class OutlineHandler(BaseHTTPRequestHandler):
    def _redirect_to_outline(self) -> None:
        if self.path != "/gabf_outline":
            self.send_error(404)
            return

        try:
            url = latest_outline_url()
        except (OSError, URLError, ValueError) as error:
            self.send_error(502, f"Unable to find the latest outline: {error}")
            return

        self.send_response(302)
        self.send_header("Location", url)
        self.send_header("Cache-Control", "no-store")
        self.end_headers()

    def do_GET(self) -> None:  # noqa: N802 (required by BaseHTTPRequestHandler)
        self._redirect_to_outline()

    def do_HEAD(self) -> None:  # noqa: N802 (required by BaseHTTPRequestHandler)
        self._redirect_to_outline()

    def log_message(self, format: str, *args: object) -> None:
        # systemd/journald records the normal request log when needed.
        return


if __name__ == "__main__":
    ThreadingHTTPServer((LISTEN_ADDRESS, LISTEN_PORT), OutlineHandler).serve_forever()
