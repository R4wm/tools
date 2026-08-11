# get_outline_references

`get_outline_references` extracts Bible references from a Grace Ambassadors
outline PDF. The current outline is available through the stable
`https://prsmusa.com/gabf_outline` endpoint, which redirects to the newest
valid outline listed at [Grace Ambassadors audio](https://graceambassadors.com/audio/).

``` bash
get_outline_references $(curl -Ls -o /dev/null -w %{url_effective} "https://prsmusa.com/gabf_outline")
```

The redirect service is implemented by `gabf_outline_redirect.py` and managed
on the web host with `gabf-outline.service`. It listens only on localhost; the
public route is provided by Nginx.
