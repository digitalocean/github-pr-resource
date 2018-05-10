## Github PR resource (WIP)

Concourse resource for Github Pull Requests written in Go and based on the [Github GraphQL API](https://developer.github.com/v4/object/commit/).
Using [the original](https://github.com/jtarchie/github-pullrequest-resource) as inspiration, but will
be made for use with webhooks, and hopefully be a bit simpler/bare bones than the original.

## Source Configuration

- `repository`: `owner/repository-name` this resource should target.
- `access_token`: A Github Access Token with repository access (required for setting status on commits).
- `path`: Only produce new versions if the PR includes changes to files that match a glob pattern.
- `ignore_path`: Inverse of the above.
- `disable_ci_skip`: Disable ability to skip builds with `[ci skip]` and `[skip ci]` in commit message.

## Behaviour

#### `check`

Produces new versions for all commits (after the last version) ordered by the push date.
A version is represented as follows:

- `pr`: The subject ID of the pull request.
- `commit`: The subject ID of the last commit on the Pullrequest.
- `pushed`: Timestamp of when the commit was pushed (and webhook triggered). Used to filter subsequent checks.

If several commits are pushed to a given PR at the same time, the last commit will be the new version.

#### `get`

- `fetch_merge`: *Optional*: Fetches the branch and merges master into it. Defaults to `true`. 

#### `put`

- `path`: The name given to the resource in a GET step.
- `status`: *Optional*: One of `SUCCESS`, `PENDING`, `FAILURE` and `ERROR`.
- `context`: *Optional*: A context to use for the status. (Prefixed with `concourse-ci`).
- `comment`: *Optional*: A comment to add to the pull request.
- `comment_file`: *Optional*: Path to file containing a comment to add to the pull request (e.g. output of `terraform plan`).

Note that `comment` and `comment_file` will be added as separate comments.

## Costs

The Github API(s) have a rate limit of 5000 requests per hour (per user). This
resource will incur the following costs:

- `check`:
  - Minimum 1, max 1 per 100th open pull request.
  - When using `path`/`ignore_path`: Minimum 1 request per *new* commit, more if the commit contains more than 100 changed files.
  - NOTE: From my experiments it seems like requests to the V3 API does not count toward the rate limit of the V4 API and vice versa.
- `in`: Minimum 1 (get information for the clone).
- `out`: Minimum 1, max 3 (1 per `status`, `comment` and `comment_file`).

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
