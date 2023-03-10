interface RuntimeReaderHandler {
  readonly bufferSize: number;
  readInto(
    buffer: ArrayBuffer,
    offset: number,
    length: number
  ): Promise<number>;
}

const __runtimeIOReaderWrapper = (
  goWrapper: RuntimeReaderHandler
): ReadableStream => {
  const stream = new ReadableStream({
    type: "bytes",
    autoAllocateChunkSize: goWrapper.bufferSize,
    async pull(controller: ReadableByteStreamController) {
      if (!controller.byobRequest || !controller.byobRequest.view) {
        throw new Error(
          "Runtime error: byobRequest or byobRequest.view is null"
        );
      }
      const view = controller.byobRequest.view;
      const read = await goWrapper.readInto(
        view.buffer,
        view.byteOffset,
        view.byteLength
      );
      if (read === 0) {
        controller.close();
      }
      controller.byobRequest.respond(read);
    },
    cancel() {},
  });
  // save the reference of NativeReaderWrapper,
  // needed for Fetcher
  (stream as any).wrapper = goWrapper;
  return stream;
};
