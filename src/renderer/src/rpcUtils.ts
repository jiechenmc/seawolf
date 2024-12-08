const PORT = 8081

export async function registerUser(
  username: string,
  password: string,
  seed: string,
  id: number = 1
): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_register',
    params: [username, password, seed]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.result !== 'sucess') {
    throw new Error('Error registering new user: ', data.error)
  }

  return data.result
}

export async function loginUser(username: string, password: string, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_login',
    params: [username, password]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.result !== 'sucess') {
    throw new Error('Error logging in user: ', data.error)
  }

  return data.result
}
