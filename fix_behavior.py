with open('web/default/src/features/system-settings/general/system-behavior-section.tsx') as f:
    content = f.read()

# Find the third conflict block
start_marker = '<<<<<<< HEAD\n\n          <FormField\n            control={form.control}\n            name=\'ForwardClientXRequestIdEnabled\''
end_marker = '>>>>>>> github/main\n      </Form>'

start_idx = content.find(start_marker)
if start_idx == -1:
    print("ERROR: Could not find start")
    exit(1)

end_idx = content.find(end_marker, start_idx)
if end_idx == -1:
    print("ERROR: Could not find end")
    exit(1)

end_idx += len(end_marker)

# Replacement: keep user's FormFields, remove Button, close with </SettingsForm>
replacement = """          <FormField
            control={form.control}
            name='ForwardClientXRequestIdEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Forward client X-Request-Id to upstream')}
                  </FormLabel>
                  <FormDescription>
                    {t('Use the client-provided X-Request-Id and pass it to upstream providers')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='ForwardUpstreamXRequestIdEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Forward upstream X-Request-Id to client')}
                  </FormLabel>
                  <FormDescription>
                    {t('Return the upstream X-Request-Id header back to the client response when available')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='UserLogContentEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Allow users to view log conversation content')}
                  </FormLabel>
                  <FormDescription>
                    {t('Control whether users can see prompt and reply content in usage log details')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='AdminLogContentEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-4'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-base'>
                    {t('Allow admins to view log conversation content')}
                  </FormLabel>
                  <FormDescription>
                    {t('Control whether admins can see prompt and reply content in usage log details')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>"""

new_content = content[:start_idx] + replacement + content[end_idx:]

with open('web/default/src/features/system-settings/general/system-behavior-section.tsx', 'w') as f:
    f.write(new_content)

print("system-behavior-section.tsx conflict 3 resolved")
