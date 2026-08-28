import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const manifestURL = new URL("../manifest.json", import.meta.url);
const pngSignature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

async function readPNGSize(relativePath) {
  const bytes = await readFile(new URL(`../${relativePath}`, import.meta.url));
  assert.ok(bytes.subarray(0, 8).equals(pngSignature), `${relativePath} is not a PNG`);
  assert.equal(bytes.subarray(12, 16).toString("ascii"), "IHDR", `${relativePath} has no IHDR`);
  return { width: bytes.readUInt32BE(16), height: bytes.readUInt32BE(20) };
}

test("manifest icons exist and match every declared pixel size", async () => {
  const manifest = JSON.parse(await readFile(manifestURL, "utf8"));

  for (const [sizeText, relativePath] of Object.entries(manifest.icons)) {
    const size = Number(sizeText);
    assert.deepEqual(await readPNGSize(relativePath), { width: size, height: size });
  }

  for (const [sizeText, relativePath] of Object.entries(manifest.action.default_icon)) {
    assert.equal(relativePath, manifest.icons[sizeText]);
  }
});
