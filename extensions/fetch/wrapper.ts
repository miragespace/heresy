declare function fetch(input: Request | RequestInit, options?: Request | RequestInit): Promise<Response>

const __runtimeFetch = (): typeof fetch => {
    return async (): Promise<Response> => {
        return new Response()
    }
}