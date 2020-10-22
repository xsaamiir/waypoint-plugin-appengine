# Waypoint Plugin Google App Engine

waypoint-plugin-appengine is a deploy (platform & release) plugin for [Waypoint](https://github.com/hashicorp/waypoint).
It allows you to stage previously built zip artifcats to Google App Engine and then release the staged deployment and
open it to general traffic.

**The plugin works but as expected for my use case but is still missing some features, please open an issue for any
feedback, issues or missing features.**

# Current limitations

- Only works with Google App Engine Standard Environment
- Only tested with an already deployed applications and services, I am not sure it works deploying a new app/service
  from scratch.

# Install

To install the plugin, run the following command:

````bash
git clone git@github.com:sharkyze/waypoint-plugin-appengine.git # or gh repo clone sharkyze/waypoint-plugin-appengine
cd waypoint-plugin-appengine
make install
````

# Authentication

Please follow the instructions in
the [Google Cloud Run tutorial](https://learn.hashicorp.com/tutorials/waypoint/google-cloud-run?in=waypoint/deploy-google-cloud#authenticate-to-google-cloud)
. This plugin uses GCP Application Default Credentials (ADC) for authentication. More
info [here](https://cloud.google.com/docs/authentication/production).

# Configure

```hcl
project = "project-name"

app "webapp" {
  path = "./webapp"

  url {
    auto_hostname = false
  }

  build {
    use "archive" {
      ignore = ["node_modules", ".git"]
    }

    registry {
      use "cloudstorage" {
        name = "artifcats/webapp/${gitrefpretty()}.zip"
        bucket = "staging.project-name.appspot.com"
      }
    }

    deploy {
      use "appengine" {
        project = "project_id"
        service = "api"
        runtime = "nodejs12"
        instance_class = "F1"
        automatic_scaling {
          max_instances = 1
        }
        main = "github.com/org/project/cmd/api"
        environment_variables = {
          "PORT": "8080"
          "SECRET_NAME_DB_URL": "projects/project-name/secrets/postgres-url/versions/latest"
        }
        handlers {
          url = "/"
          static_files = "build/index.html"
          upload = "build/index.html"
          secure = "SECURE_ALWAYS"
          http_headers = {
            "Strict-Transport-Security": "max-age=31536000; includeSubDomains"
          }
        }
        handlers {
          url = "/.*"
          static_files = "build/index.html"
          upload = "build/index.html"
          secure = "SECURE_ALWAYS"
          http_headers = {
            "Strict-Transport-Security": "max-age=31536000; includeSubDomains"
          }
        }
      }
    }

    release {
      use "appengine" {}
    }
  }
}
```
