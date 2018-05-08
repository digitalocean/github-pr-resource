## Github PR resource (WIP)

Concourse resource for Github Pull Requests written in Go and based on the [Github GraphQL API](https://developer.github.com/v4/object/commit/).
Using [the original](https://github.com/jtarchie/github-pullrequest-resource) as inspiration, but will
be made for use with webhooks, and hopefully be a bit simpler/bare bones than the original.

## Source Configuration

- `repository`: `owner/repository-name` this resource should target.
- `access_token`: A Github Access Token with repository access (required for setting status on commits).

## Behaviour

#### `check`

Produces new versions for all commits (after the last version) ordered by the push date.

#### `get`

- `fetch_merge`: *Optional*: Fetches the branch and merges master into it. Defaults to `true`. 

#### `put`

- `status`: One of `SUCCESS`, `PENDING`, `FAILURE` and `ERROR`.
- `context`: A context to use for the status.
- `comment`: Add a comment to the PR.

TODO

## Example

The following (incomplete) example would build a new AMI using Packer:

```yaml
resource_types:
- name: pull-request
  type: docker-image
  source:
    repository: itsdalmo/github-pr-resource

resources:
- name: test-repository-pr
  type: pull-request 
  source:
    repository: itsdalmo/test-repository
    access_token: ((github-access-token))
    context: concourse-ci/status

jobs:
- name: test-pr
  plan:
  - get: test-repository-pr
```
