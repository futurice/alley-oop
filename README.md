# Alley-Oop

For information about what is an alley-oop, please refer to [this instructional
video](https://www.youtube.com/watch?v=cpfCd9fWq04).

In this context, Alley-Oop is a Dynamic DNS server that manages also fetching
HTTPS certificates automatically for the domains, making it possible to use them
in a local area network in addition to the public internet.

## Release

1. Make sure all changes are pushed to GitHub `master`
1. Go on [GitHub](https://github.com/futurice/alley-oop/releases) and draft a new release with the format `v1.0.0`
1. Go on [Docker Hub](https://hub.docker.com/r/futurice/alley-oop/~/settings/automated-builds/), update the tag name, save, and use the "Trigger" button:
  ![Docker Hub build](doc/docker-hub-build.png)
