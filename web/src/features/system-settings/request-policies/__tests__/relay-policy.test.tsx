import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'

import { GroupRelayPolicyEditor } from '@/features/system-settings/models/group-relay-policy-editor'

function PolicyEditorFixture() {
  const [value, setValue] = useState(
    '{"default":{"endpoint":"/","require_key":true,"auto_route":true,"route_strategy":"sequential","fallback_times":1}}'
  )

  return (
    <>
      <GroupRelayPolicyEditor
        value={value}
        groupOptions={['default', 'vip']}
        modelOptions={['model-a', 'model-b']}
        onChange={setValue}
      />
      <output aria-label='Serialized policies'>{value}</output>
    </>
  )
}

describe('relay access policy editor', () => {
  it('saves fallback count, target group, and selected model', async () => {
    const user = userEvent.setup()
    render(<PolicyEditorFixture />)

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Fallback attempts' }), {
      target: { value: '2' },
    })
    await user.click(screen.getByRole('combobox', { name: 'Fallback group' }))
    await user.click(await screen.findByRole('option', { name: 'vip' }))
    await user.click(screen.getByRole('combobox', { name: 'Fallback model' }))
    await user.click(await screen.findByRole('option', { name: 'model-b' }))

    expect(JSON.parse(screen.getByLabelText('Serialized policies').textContent ?? '{}')).toMatchObject({
      default: {
        fallback_times: 2,
        fallback_group: 'vip',
        fallback_model: 'model-b',
      },
    })
  })
})
