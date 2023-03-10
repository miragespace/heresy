async function drainStream(stream) {
  const chunks = [];
  const reader = stream.getReader();

  function readNextChunk() {
    return reader.read().then(({ done, value }) => {
      if (done) {
        return chunks.reduce((bytes, chunk) => [...bytes, ...chunk], []);
      }

      chunks.push(value);

      return readNextChunk();
    });
  }

  const bytes = await readNextChunk();

  return new Uint8Array(bytes);
}