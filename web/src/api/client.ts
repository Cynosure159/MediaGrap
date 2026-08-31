export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...options,
  })

  if (!response.ok) {
    const data = await response.json().catch(() => null) as { error?: { message?: string } } | null
    throw new Error(data?.error?.message ?? `Request failed with status ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}
