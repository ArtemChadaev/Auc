import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from "~/components/ui/card";
import {Button} from "~/components/ui/button";
import {Field, FieldDescription, FieldLabel} from "~/components/ui/field";
import {Input} from "~/components/ui/input";

const StepRegister = () => {
  return (
    <Card className="mx-auto max-w-md w-xs">
      <CardHeader>
        <CardTitle>Write your email</CardTitle>
        <CardDescription>
          A verification code will be sent to your email address.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Field>
          <div className="flex items-center justify-between">
            <FieldLabel htmlFor="otp-verification">
              Email
            </FieldLabel>
          </div>
          <Input placeholder="user@example.com"/>
          <FieldDescription>
            <a href="/auth">I have account</a>
          </FieldDescription>
        </Field>
      </CardContent>
      <CardFooter>
        <Field>
          <Button className="w-full">
            Send code
          </Button>
        </Field>
      </CardFooter>
    </Card>
  )
}

export default StepRegister