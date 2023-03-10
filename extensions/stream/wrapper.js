const __runtimeIOReaderWrapper = (goWrapper) => {
    return new ReadableStream({
        type: "bytes",
        autoAllocateChunkSize: goWrapper.size(),
        async pull(controller) {
            const view = controller.byobRequest.view
            const read = await goWrapper.readInto(view.buffer, view.byteOffset, view.byteLength)
            if (read === 0) {
                goWrapper.close()
                controller.close()
            }
            controller.byobRequest.respond(read)
        },
        cancel() {
            goWrapper.close()
        }
    })
}