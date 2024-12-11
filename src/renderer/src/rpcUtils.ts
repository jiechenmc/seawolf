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

  if (data.result !== 'success') {
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

  if (data.error) {
    throw new Error('Error logging in user: ', data.error)
  }

  return data.result
}

export async function logoutUser(id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_logout',
    params: []
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error logging out user: ', data.error)
  }

  return data.result
}

export async function uploadFile(filePath: string, fileCost: number, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_putFile',
    params: [filePath, fileCost]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error uploading file: ', data.error)
  }

  return data.result
}

export async function deleteFile(cid: string, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_deleteFile',
    params: [cid]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error getting uploaded files: ', data.error)
  }

  return data.result
}

export async function getUploadedFiles(id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_getUploads',
    params: []
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error getting uploaded files: ', data.error)
  }

  return data.result
}

export async function getDownloadedFiles(id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_getDownloads',
    params: []
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error getting downloaded files: ', data.error)
  }

  return data.result
}

export async function discoverFile(cid: string, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_discoverFile',
    params: [cid]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error getting providers for a file: ', data.error)
  }

  return data.result
}

export async function getFile(
  peer_id: string,
  cid: string,
  download_path: string,
  id: number = 1
): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_getFile',
    params: [peer_id, cid, download_path]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error downloading a file: ', data.error)
  }

  return data.result
}

export async function pauseDownload(session_id: number, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_getFile',
    params: [session_id]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error pausing download: ', data.error)
  }

  return data.result
}

export async function resumeDownload(session_id: number, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_getFile',
    params: [session_id]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error resuming download: ', data.error)
  }

  return data.result
}

export async function getSessionInfo(session_id: number, id: number = 1): Promise<any> {
  const request = {
    jsonrpc: '2.0',
    id: id,
    method: 'p2p_getSession',
    params: [session_id]
  }

  const response = await fetch(`http://localhost:${PORT}/rpc`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(request)
  })

  const data = await response.json()

  if (data.error) {
    throw new Error('Error getting session info : ', data.error)
  }

  return data.result
}
