# Waypoint Plugin Google App Engine [**WIP**]

waypoint-plugin-gae is a deploy (platform & release) plugin for [Waypoint](https://github.com/hashicorp/waypoint). 
It allows you to stage previously built zip artifcats to Google App Engine and then release the staged deployment and open it to general traffic.

**This plugin is still work in progress, please open an issue for any feedback or issues.**

# Install
To install the plugin, run the following command:

````bash
git clone git@github.com:sharkyze/waypoint-plugin-gae.git # or gh repo clone sharkyze/waypoint-plugin-gcs
cd waypoint-plugin-gae
make install # Installs the plugin in `${HOME}/.config/waypoint/plugins/`
````

# GAE Authentication
Please follow the instructions in the [Google Cloud Run tutorial](https://learn.hashicorp.com/tutorials/waypoint/google-cloud-run?in=waypoint/deploy-google-cloud#authenticate-to-google-cloud).
This plugin uses GCP Application Default Credentials (ADC) for authentication. More info [here](https://cloud.google.com/docs/authentication/production).

# Configure
```hcl
project = "project"

app "webapp" {
  path = "./webapp"

  build {
    use "archive" {
      sources = ["src/", "public/", "package.json"] # Sources are relative to /path/to/project/webapp/
      output_name = "webapp.zip"
      overwrite_existing = true
    }

    registry {
      use "gcs" {
        source = "webapp.zip"
        name = "artifcats/webapp/${gitrefpretty()}.zip"
        bucket = "staging.gcp-project-name.appspot.com"
      }
    }

    deploy {
      use "gae" {
        application = "gae-app-name"
        service = "api"
        runtime = "go114"
        instance_class = "F1"
        main = "github.com/org/project/cmd/api"
        environment_variables = {
          "PORT": "8080"
        }
      }
    }
  }
}
```
