console.log(Object.getOwnPropertyNames(globalThis))

async function drainStream(stream) {
  const chunks = [];
  const reader = stream.getReader();

  async function readNextChunk(){
    const { done, value } = await reader.read();
    if (done) {
      return chunks.reduce((bytes, chunk) => [...bytes, ...chunk], []);
    }
    chunks.push(value);
    return readNextChunk();
  }

  const bytes = await readNextChunk();

  return new Uint8Array(bytes);
}