#!/usr/bin/env python3
"""Exercise CI scripts with real archives and mocked external services."""

import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parents[1]
IMAGE_REPO = "index.docker.io/mtlynch/picoshare"
AMD64_DIGEST = "sha256:" + "a" * 64
ARM64_DIGEST = "sha256:" + "b" * 64
ARMV7_DIGEST = "sha256:" + "c" * 64
INDEX_DIGEST = "sha256:" + "d" * 64

# These executables enforce the external command contracts, not just exit codes.
MOCK_COMMAND = r'''
import json
import os
from pathlib import Path
import sys
import tarfile

command = Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ["EVENTS"], "a") as events:
    events.write(json.dumps({"command": command, "args": args,
                             "cwd": os.getcwd(), "home": os.environ.get("HOME"),
                             "docker_config": os.environ.get("DOCKER_CONFIG")}) + "\n")

operation = command + " " + " ".join(args[:2])
if command == "crane" and args[0] in ("push", "manifest"):
    operation = command + " " + args[0]
if command == "skopeo":
    operation = "skopeo copy"
if os.environ.get("MOCK_FAIL") == operation:
    sys.exit(23)

repo = "index.docker.io/mtlynch/picoshare"
amd64 = "sha256:" + "a" * 64
arm64 = "sha256:" + "b" * 64
armv7 = "sha256:" + "c" * 64
index = "sha256:" + "d" * 64

if command == "nix":
    if args == ["config", "show", "system"]:
        print("aarch64-linux")
    elif args[:2] == ["flake", "check"]:
        pass
    else:
        assert args == ["copy", ".#checks.aarch64-linux.all", "--to",
                        "https://cache.nix-ci.com?compression=xz&parallel-compression=true"], args
elif command == "git":
    if args == ["rev-parse", "--short", "HEAD"]:
        print("fixture-tag")
        sys.exit(0)
    assert args in (["rev-parse", "HEAD"],
                    ["rev-parse", "refs/tags/v1.2.3^{commit}"]), args
    if args[-1] == "HEAD":
        print(os.environ.get("MOCK_HEAD", "release-commit"))
    else:
        print(os.environ.get("MOCK_TAG", "release-commit"))
elif command == "gh":
    if args[:2] == ["release", "view"]:
        assert args == ["release", "view", "v1.2.3", "--repo", "mtlynch/picoshare"], args
    else:
        assert args[:3] == ["release", "upload", "v1.2.3"], args
        assert args[-3:] == ["--repo", "mtlynch/picoshare", "--clobber"], args
        assert sorted(Path(path).name for path in args[3:-3]) == [
            "picoshare-v1.2.3-linux-amd64.tar.gz",
            "picoshare-v1.2.3-linux-arm64.tar.gz",
            "picoshare-v1.2.3-linux-armv7.tar.gz"], args
        for path in args[3:-3]:
            with tarfile.open(path, "r:gz") as archive:
                assert archive.getnames() == ["picoshare"]
elif command == "crane":
    if args[:2] == ["auth", "login"]:
        assert args == ["auth", "login", "index.docker.io", "--username", "release-user",
                        "--password-stdin"], args
        assert sys.stdin.read() == "docker-secret"
        assert Path(os.environ["DOCKER_CONFIG"]).is_dir()
        assert Path(os.environ["DOCKER_CONFIG"]).stat().st_mode & 0o777 == 0o700
    elif args[0] == "push":
        assert len(args) == 3, args
        arch = args[2].removeprefix(repo + ":v1.2.3-")
        assert arch in ("amd64", "arm64", "armv7"), args
        # An image stream executable is not a Docker archive. Opening the
        # archive catches passing that executable directly to crane push.
        with tarfile.open(args[1], "r:") as archive:
            assert archive.getnames() == ["manifest.json", "image-" + arch]
            assert json.load(archive.extractfile("manifest.json")) == [{
                "Config": "image-" + arch, "RepoTags": [], "Layers": []}]
            assert archive.extractfile("image-" + arch).read() == arch.encode()
    elif args[0] == "mutate":
        assert args == ["mutate", repo + ":v1.2.3-armv7", "--set-platform",
                        "linux/arm/v7", "--tag", repo + ":v1.2.3-armv7"], args
    elif args[0] == "digest":
        assert len(args) == 2, args
        print({repo + ":v1.2.3-amd64": amd64,
               repo + ":v1.2.3-arm64": arm64,
               repo + ":v1.2.3-armv7": armv7,
               repo + ":v1.2.3": index}[args[1]])
    elif args[:2] == ["index", "append"]:
        # Append starts a new index: no positional base image is allowed.
        assert args == ["index", "append", "--manifest", repo + "@" + amd64,
                        "--manifest", repo + "@" + arm64,
                        "--manifest", repo + "@" + armv7,
                        "--tag", repo + ":v1.2.3"], args
    elif args[0] == "manifest":
        assert args == ["manifest", repo + "@" + index], args
        platforms = [{"os": "linux", "architecture": "arm64"},
                     {"os": "linux", "architecture": "amd64"},
                     {"os": "linux", "architecture": "arm", "variant": "v7"}]
        if os.environ.get("MOCK_BAD_PLATFORMS"):
            platforms.append({"os": "linux", "architecture": "386"})
        print(json.dumps({"manifests": [{"platform": platform} for platform in platforms]}))
    else:
        assert args == ["tag", repo + "@" + index, "latest"], args
elif command == "fly":
    if args == ["auth", "docker"]:
        home = Path(os.environ["HOME"])
        assert home != Path(os.environ["ORIGINAL_HOME"])
        assert os.environ["PATH"] == str(Path(sys.argv[0]).parent)
        config = home / ".docker" / "config.json"
        config.parent.mkdir()
        config.write_text('{"auths": {"registry.fly.io": {"auth": "fixture"}}}')
    else:
        assert args == ["deploy", "--app", "fixture-app", "--image",
                        "registry.fly.io/fixture-app:deployment-fixture-tag", "--ha=false"], args
elif command == "skopeo":
    assert args[:4] == ["--policy", os.environ["POLICY"], "copy", "--dest-authfile"], args
    assert Path(args[4]).is_file()
    assert args[5].startswith("docker-archive:"), args
    with tarfile.open(args[5].removeprefix("docker-archive:"), "r:") as archive:
        assert archive.getnames() == ["manifest.json", "image-amd64"]
    assert args[6:] == ["docker://registry.fly.io/fixture-app:deployment-fixture-tag"], args
else:
    raise AssertionError("unexpected command: " + command)
'''


class WorkflowTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="ci-workflows-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.repo = self.root / "repo"
        self.scripts = self.repo / "dev-scripts"
        self.scripts.mkdir(parents=True)
        shutil.copy2(SCRIPTS / "publish-release", self.scripts)
        shutil.copy2(SCRIPTS / "package-binaries", self.scripts)
        shutil.copy2(SCRIPTS / "build-all-flake-targets", self.scripts)
        shutil.copy2(SCRIPTS / "upload-to-nixci-cache", self.scripts)
        shutil.copy2(SCRIPTS / "deploy-to-fly", self.scripts)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        mock = "#!" + sys.executable + "\n" + MOCK_COMMAND
        self.write_executable(self.bin / "nix", mock)
        self.write_executable(self.bin / "git", mock)
        self.write_executable(self.bin / "gh", mock)
        self.write_executable(self.bin / "crane", mock)
        self.write_executable(self.bin / "fly", mock)
        self.write_executable(self.bin / "skopeo", mock)
        self.events_path = self.root / "events.jsonl"
        self.other_cwd = self.root / "unrelated"
        self.other_cwd.mkdir()
        self.temp = self.root / "tmp"
        self.temp.mkdir()
        self.env = dict(os.environ)
        self.env.update(PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        EVENTS=str(self.events_path), TMPDIR=str(self.temp), LC_ALL="C")
        self.env.pop("MOCK_FAIL", None)
        self.env.pop("MOCK_BAD_PLATFORMS", None)
        self.binaries = self.root / "binaries"
        self.images = self.root / "images"
        self.images.mkdir()
        self.create_architecture("amd64")
        self.create_architecture("arm64")
        self.create_architecture("armv7")

    def write_executable(self, path, source):
        path.write_text(source)
        path.chmod(0o755)

    def create_architecture(self, arch):
        directory = self.binaries / ("linux-" + arch)
        directory.mkdir(parents=True)
        (directory / "picoshare").write_bytes(("binary-" + arch).encode())
        (directory / "not-for-release").write_text("must not be packaged")
        image = self.root / (arch + ".tar")
        with tarfile.open(image, "w") as archive:
            manifest = json.dumps([{"Config": "image-" + arch,
                                    "RepoTags": [], "Layers": []}]).encode()
            member = tarfile.TarInfo("manifest.json")
            member.size = len(manifest)
            archive.addfile(member, io.BytesIO(manifest))
            member = tarfile.TarInfo("image-" + arch)
            member.size = len(arch)
            archive.addfile(member, io.BytesIO(arch.encode()))
        self.write_executable(self.images / ("linux-" + arch),
                              "#!" + shutil.which("bash") + "\n# Stream the fixture Docker archive.\n"
                              + "cat '" + str(image) + "'\n")

    def run_script(self, name, *args):
        return subprocess.run([str(self.scripts / name), *map(str, args)],
                              cwd=self.other_cwd, env=self.env, text=True,
                              capture_output=True, timeout=30)

    def events(self):
        if not self.events_path.exists():
            return []
        return [json.loads(line) for line in self.events_path.read_text().splitlines()]

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def release_environment(self):
        self.env.update(NIX_CI_GIT_REF="refs/tags/v1.2.3",
                        NIX_CI_GIT_COMMIT_SHA="release-commit",
                        GH_TOKEN="github-secret", DOCKERHUB_USERNAME="release-user",
                        DOCKERHUB_ACCESS_TOKEN="docker-secret")

    def publish(self):
        return self.run_script("publish-release", self.binaries, self.images,
                               self.scripts / "package-binaries")

    def assert_no_latest(self):
        self.assertNotIn(["tag", IMAGE_REPO + "@" + INDEX_DIGEST, "latest"],
                         [event["args"] for event in self.events()])

    def assert_no_release_upload(self):
        self.assertEqual([event for event in self.events()
                          if event["command"] == "gh"
                          and event["args"][:2] == ["release", "upload"]], [])

    def test_aggregate_defaults_and_upload_from_unrelated_directory(self):
        result = self.run_script("build-all-flake-targets")
        self.assert_success(result)
        events = self.events()
        self.assertEqual([event["args"] for event in events], [
            ["flake", "check", "--max-jobs", "auto"],
            ["config", "show", "system"],
            ["copy", ".#checks.aarch64-linux.all", "--to",
             "https://cache.nix-ci.com?compression=xz&parallel-compression=true"]])
        self.assertEqual([event["cwd"] for event in events], [str(self.repo)] * 3)

    def test_aggregate_forwards_custom_options_without_defaults(self):
        result = self.run_script("build-all-flake-targets", "--max-jobs", "2",
                                 "--option", "fixture option", "value with spaces")
        self.assert_success(result)
        self.assertEqual(self.events()[0]["args"], [
            "flake", "check", "--max-jobs", "2", "--option",
            "fixture option", "value with spaces"])
        self.assertEqual(self.events()[0]["cwd"], str(self.repo))

    def test_failed_build_never_uploads(self):
        self.env["MOCK_FAIL"] = "nix flake check"
        result = self.run_script("build-all-flake-targets")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual([event["args"] for event in self.events()],
                         [["flake", "check", "--max-jobs", "auto"]])

    def test_failed_cache_upload_fails_aggregate(self):
        self.env["MOCK_FAIL"] = "nix copy .#checks.aarch64-linux.all"
        result = self.run_script("build-all-flake-targets")
        self.assertEqual(result.returncode, 23, result.stderr)
        self.assertEqual(self.events()[-1]["args"][0], "copy")

    def test_upload_uses_current_system_and_repo_from_unrelated_directory(self):
        result = self.run_script("upload-to-nixci-cache")
        self.assert_success(result)
        self.assertEqual([event["args"] for event in self.events()], [
            ["config", "show", "system"],
            ["copy", ".#checks.aarch64-linux.all", "--to",
             "https://cache.nix-ci.com?compression=xz&parallel-compression=true"]])
        self.assertEqual([event["cwd"] for event in self.events()], [str(self.repo)] * 2)

    def assert_binary_archive(self, path, contents):
        self.assertEqual(path.read_bytes()[:2], b"\x1f\x8b")
        with tarfile.open(path, "r:gz") as archive:
            self.assertEqual(archive.getnames(), ["picoshare"])
            self.assertTrue(archive.getmember("picoshare").isfile())
            self.assertEqual(archive.extractfile("picoshare").read(), contents)

    def test_packager_creates_exact_archives_with_only_each_binary(self):
        output = self.root / "release output"
        result = self.run_script("package-binaries", "1.2.3", self.binaries, output)
        self.assert_success(result)
        self.assertEqual(sorted(path.name for path in output.iterdir()), [
            "picoshare-v1.2.3-linux-amd64.tar.gz",
            "picoshare-v1.2.3-linux-arm64.tar.gz",
            "picoshare-v1.2.3-linux-armv7.tar.gz"])
        self.assert_binary_archive(output / "picoshare-v1.2.3-linux-amd64.tar.gz", b"binary-amd64")
        self.assert_binary_archive(output / "picoshare-v1.2.3-linux-arm64.tar.gz", b"binary-arm64")
        self.assert_binary_archive(output / "picoshare-v1.2.3-linux-armv7.tar.gz", b"binary-armv7")

    def test_non_tag_is_rejected_before_credentials_or_external_commands(self):
        self.env["NIX_CI_GIT_REF"] = "refs/heads/master"
        self.env.pop("GH_TOKEN", None)
        result = self.publish()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("must be a semantic-version release tag", result.stderr)
        self.assertEqual(self.events(), [])

    def test_wrong_head_is_rejected_before_credentials_or_network(self):
        self.release_environment()
        self.env["MOCK_HEAD"] = "wrong-commit"
        self.env.pop("GH_TOKEN")
        result = self.publish()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("HEAD and release tag must match", result.stderr)
        self.assertEqual([event["command"] for event in self.events()], ["git", "git"])

    def test_wrong_tag_commit_is_rejected_before_credentials_or_network(self):
        self.release_environment()
        self.env["MOCK_TAG"] = "wrong-commit"
        self.env.pop("GH_TOKEN")
        result = self.publish()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("HEAD and release tag must match", result.stderr)
        self.assertEqual([event["command"] for event in self.events()], ["git", "git"])

    def test_publish_streams_archives_builds_immutable_index_and_uploads_before_latest(self):
        self.release_environment()
        self.env["LC_ALL"] = "en_US.UTF-8"
        result = self.publish()
        self.assert_success(result)
        events = self.events()
        self.assertEqual([event["args"] for event in events
                          if event["command"] == "crane" and event["args"][0] == "mutate"], [
            ["mutate", IMAGE_REPO + ":v1.2.3-armv7", "--set-platform",
             "linux/arm/v7", "--tag", IMAGE_REPO + ":v1.2.3-armv7"]])
        self.assertEqual([event["args"] for event in events
                          if event["command"] == "crane" and event["args"][:2] == ["index", "append"]], [
            ["index", "append", "--manifest", IMAGE_REPO + "@" + AMD64_DIGEST,
             "--manifest", IMAGE_REPO + "@" + ARM64_DIGEST,
             "--manifest", IMAGE_REPO + "@" + ARMV7_DIGEST,
             "--tag", IMAGE_REPO + ":v1.2.3"]])
        self.assertEqual(events[-3]["args"], ["manifest", IMAGE_REPO + "@" + INDEX_DIGEST])
        self.assertEqual(events[-2]["command"], "gh")
        self.assertEqual(events[-2]["args"][:3], ["release", "upload", "v1.2.3"])
        self.assertEqual(events[-1]["args"], ["tag", IMAGE_REPO + "@" + INDEX_DIGEST, "latest"])
        self.assertFalse(Path(events[-1]["docker_config"]).exists())
        self.assertEqual(list(self.temp.iterdir()), [])

    def test_failed_image_push_never_uploads_or_updates_latest(self):
        self.release_environment()
        self.env["MOCK_FAIL"] = "crane push"
        result = self.publish()
        self.assertEqual(result.returncode, 23, result.stderr)
        self.assert_no_release_upload()
        self.assert_no_latest()
        self.assertEqual(list(self.temp.iterdir()), [])

    def test_failed_github_upload_never_updates_latest(self):
        self.release_environment()
        self.env["MOCK_FAIL"] = "gh release upload"
        result = self.publish()
        self.assertEqual(result.returncode, 23, result.stderr)
        self.assert_no_latest()
        self.assertEqual(self.events()[-1]["args"][:2], ["release", "upload"])
        self.assertEqual(list(self.temp.iterdir()), [])

    def test_failed_manifest_fetch_never_uploads_or_updates_latest(self):
        self.release_environment()
        self.env["MOCK_FAIL"] = "crane manifest"
        result = self.publish()
        self.assertNotEqual(result.returncode, 0)
        self.assert_no_release_upload()
        self.assert_no_latest()

    def test_unexpected_manifest_platform_never_uploads_or_updates_latest(self):
        self.release_environment()
        self.env["MOCK_BAD_PLATFORMS"] = "1"
        result = self.publish()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("platforms did not match expected platforms", result.stderr)
        self.assert_no_release_upload()
        self.assert_no_latest()

    def fly_environment(self):
        shutil.copy2(self.images / "linux-amd64", self.bin / "stream-docker-image")
        self.policy = self.root / "policy.json"
        self.policy.write_text('{"default": [{"type": "reject"}]}')
        self.env.update(FLY_API_TOKEN="fly-secret", POLICY=str(self.policy),
                        ORIGINAL_HOME=self.env["HOME"])
        (self.other_cwd / "fly.toml").write_text('app = "fixture-app"\n')

    def deploy(self):
        return self.run_script("deploy-to-fly", self.policy)

    def test_fly_uses_explicit_policy_and_removes_temporary_auth_after_success(self):
        self.fly_environment()
        result = self.deploy()
        self.assert_success(result)
        events = self.events()
        self.assertEqual([event["command"] for event in events], ["git", "fly", "skopeo", "fly"])
        self.assertFalse(Path(events[1]["home"]).exists())
        self.assertEqual(events[2]["args"][:4], ["--policy", str(self.policy), "copy", "--dest-authfile"])
        self.assertEqual(list(self.temp.iterdir()), [])

    def test_fly_removes_temporary_auth_and_never_deploys_after_copy_failure(self):
        self.fly_environment()
        self.env["MOCK_FAIL"] = "skopeo copy"
        result = self.deploy()
        self.assertEqual(result.returncode, 23, result.stderr)
        events = self.events()
        self.assertEqual([event["command"] for event in events], ["git", "fly", "skopeo"])
        self.assertFalse(Path(events[1]["home"]).exists())
        self.assertEqual(list(self.temp.iterdir()), [])


if __name__ == "__main__":
    unittest.main()
