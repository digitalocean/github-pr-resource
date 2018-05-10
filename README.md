## Github PR resource (WIP)

Concourse resource for Github Pull Requests written in Go and based on the [Github GraphQL API](https://developer.github.com/v4/object/commit/).
Using [the original](https://github.com/jtarchie/github-pullrequest-resource) as inspiration, but will
be made for use with webhooks, and hopefully be a bit simpler/bare bones than the original.

## Source Configuration

- `repository`: `owner/repository-name` this resource should target.
- `access_token`: A Github Access Token with repository access (required for setting status on commits).
- `path`: Only produce new versions if the PR includes changes to files that match a glob pattern.
- `ignore_path`: Inverse of the above.

## Behaviour

#### `check`

Produces new versions for all commits (after the last version) ordered by the push date.

#### `get`

- `fetch_merge`: *Optional*: Fetches the branch and merges master into it. Defaults to `true`. 

#### `put`

- `path`: The name given to the resource in a GET step.
- `status`: *Optional*: One of `SUCCESS`, `PENDING`, `FAILURE` and `ERROR`.
- `context`: *Optional*: A context to use for the status. (Prefixed with `concourse-ci`).
- `comment`: *Optional*: A comment to add to the pull request.
- `comment_file`: *Optional*: Path to file containing a comment to add to the pull request (e.g. output of `terraform plan`).

Note that `comment` and `comment_file` will be added as separate comments.

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
