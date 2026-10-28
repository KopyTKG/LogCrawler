import { render, screen } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'
import Detail from './Detail.svelte'
import type { FileInfo } from '../../bindings/logcrawler'

const file = {
  path: '/x.log',
  name: 'x.log',
  title: 'DRIVE! Example',
  header: [{ key: 'Variant', value: 'exampletest' }],
  sessions: [],
  problems: [],
  tags: [],
  lines: 10,
} as FileInfo

describe('Detail', () => {
  it('shows the JSON of a failure body as its own block', () => {
    render(Detail, {
      text: '10:00:00.000 LOG   [http] #7 ✕ body {"status":500}',
      node: null,
      file,
    })
    expect(screen.getByText('Response body')).toBeInTheDocument()
    expect(screen.getByText('"status":')).toHaveClass('tk-key')
  })

  it('shows the file header on a session', () => {
    render(Detail, {
      text: 'previous session',
      node: { kind: 'session' } as never,
      file,
    })
    expect(screen.getByText('exampletest')).toBeInTheDocument()
  })

  it('shows earlier events before the selection', () => {
    render(Detail, {
      text: '10:00:02.000 LOG   selected',
      before: '10:00:01.000 LOG   earlier',
      node: null,
      file,
    })
    expect(screen.getByText(/1 earlier events/)).toBeInTheDocument()
  })
})
