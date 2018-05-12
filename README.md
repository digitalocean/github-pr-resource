## Github PR resource

Concourse resource for Github Pull Requests written in Go and based on the [Github GraphQL API](https://developer.github.com/v4/object/commit/).
It is based on [jtarchie/github-pullrequest-resource](https://github.com/jtarchie/github-pullrequest-resource), with some important differences:

- Uses the GraphQL API for `check`, which means that a `check` only requires 1 API call per 100th open pull request.
- Because `check` uses GraphQL we can afford (see [costs](#costs)) to not use any caching, which means this resource also plays nice with webhooks.
- `get` always clones master and merges the PR up until a given (by version) commit.
- Does not require two `get` steps to `fetch_merge` and still set status on the correct commit using `put`.
- ... not as many features (yet!).

## Source Configuration

|     Parameter     | Required |          Example           |                                                             Description                                                              |
| ----------------- | -------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `repository`      | Yes      | `itsdalmo/test-repository` | The repository to target.                                                                                                            |
| `access_token`    | Yes      |                            | A Github Access Token with repository access (required for setting status on commits).                                               |
| `path`            | No       | `terraform/*.tf`           | Only produce new versions if the PR includes changes to files that match a [path.Match](https://golang.org/pkg/path/#Match) pattern. |
| `ignore_path`     | No       | `.ci/*`                    | Inverse of the above.                                                                                                                |
| `disable_ci_skip` | No       | `true` (string)            | Disable ability to skip builds with `[ci skip]` and `[skip ci]` in commit message or pull request title.                             |


## Behaviour

#### `check`

Produces new versions for all commits (after the last version) ordered by the push date.
A version is represented as follows:

- `pr`: The subject ID of the pull request.
- `commit`: The subject ID of the last commit on the Pullrequest.
- `pushed`: Timestamp of when the commit was pushed (and webhook triggered). Used to filter subsequent checks.

If several commits are pushed to a given PR at the same time, the last commit will be the new version.

#### `get`

Clones the base (e.g. `master` branch) at the latest commit, and merges the pull request at the specified commit
into master. This ensures that we are both testing and setting status on the exact commit that was requested in
input. Because the base of the PR is not locked to a specific commit in versions emitted from `check`, a fresh
`get` will always use the latest commit in master and *report the SHA of said commit in the metadata*.

Note that, should you retrigger a build in the hopes of testing the last commit to a PR against a newer version of
the base, Concourse will reuse the volume (i.e. not trigger a new `get`) if it still exists, which can produce
unexpected results (#5). As such, re-testing a PR against a newer version of the base is best done by *pushing an 
empty commit to the PR*.

#### `put`

|   Parameter    | Required |         Example         |                                             Description                                             |
| -------------- | -------- | ----------------------- | --------------------------------------------------------------------------------------------------- |
| `path`         | Yes      | `pull-request`          | The name given to the resource in a GET step.                                                       |
| `status`       | No       | `SUCCESS`               | Set a status on a commit. One of `SUCCESS`, `PENDING`, `FAILURE` and `ERROR`.                       |
| `context`      | No       | `unit-test`             | A context to use for the status. (Prefixed with `concourse-ci`, defaults to `concourse-ci/status`). |
| `comment`      | No       | `hello world!`          | A comment to add to the pull request.                                                               |
| `comment_file` | No       | `my-output/comment.txt` | Path to file containing a comment to add to the pull request (e.g. output of `terraform plan`).     |

## Example

```yaml
resource_types:
- name: pull-request
  type: docker-image
  source:
    repository: itsdalmo/github-pr-resource

resources:
- name: pull-request
  type: pull-request
  check_every: 24h
  webhook_token: ((webhook-token))
  source:
    repository: itsdalmo/test-repository
    access_token: ((github-access-token))

jobs:
- name: test
  plan:
  - get: pull-request
    trigger: true
    version: every
  - put: pull-request
    params:
      path: pull-request
      status: pending
  - task: unit-test
    config:
      platform: linux
      image_resource:
        type: docker-image
        source: {repository: alpine/git, tag: "latest"}
      inputs:
        - name: pull-request
      run:
        path: /bin/sh
        args:
          - -xce
          - |
            cd pull-request
            git log --graph --all --color --pretty=format:"%x1b[31m%h%x09%x1b[32m%d%x1b[0m%x20%s" > log.txt
            cat log.txt
    on_failure:
      put: pull-request
      params:
        path: pull-request
        status: failure
  - put: pull-request
    params:
      path: pull-request
      status: success
```

## Costs

The Github API(s) have a rate limit of 5000 requests per hour (per user). This resource will incur the following costs:

- `check`: Minimum 1, max 1 per 100th *open* pull request.
- `in`: Fixed cost of 2. Fetches PR and Commit from global ID (passed in via version).
- `out`: Minimum 1, max 3 (1 for each of `status`, `comment` and `comment_file`).

E.g., typical use for a repository with 125 open pull requests will incur the following costs for every commit:

- `check`: 2 (paginate 125 PR's with 100 per page)
- `in`: 2 (fetch commit and PR from global ID's)
- `out`: 1 (set status on the commit)

With a rate limit of 5000 per hour, it could handle 1000 commits between all of the 125 open pull requests in the span of that hour.