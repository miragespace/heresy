"use strict";
const __runtimeIOReaderWrapper = (goWrapper) => {
    return new ReadableStream({
        type: "bytes",
        autoAllocateChunkSize: goWrapper.bufferSize,
        async pull(controller) {
            if (!controller.byobRequest || !controller.byobRequest.view) {
                throw new Error("Runtime error: byobRequest or byobRequest.view is null");
            }
            const view = controller.byobRequest.view;
            const read = await goWrapper.readInto(view.buffer, view.byteOffset, view.byteLength);
            if (read === 0) {
                goWrapper.close(goWrapper);
                controller.close();
            }
            controller.byobRequest.respond(read);
        },
        cancel() {
            goWrapper.close(goWrapper);
        },
    });
};
