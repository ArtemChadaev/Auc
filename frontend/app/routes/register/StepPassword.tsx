import {Button} from "~/components/ui/button";
import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from "~/components/ui/card";
import {Field} from "~/components/ui/field";
import {Label} from "~/components/ui/label";
import {Input} from "~/components/ui/input";

const StepPassword = () => {
  return (
    <Card className="mx-auto max-w-md w-xs">
      <CardHeader>
        <CardTitle>Use science where write password</CardTitle>
        <CardDescription>
          Or write something.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Field>
          <Label htmlFor="PasswordRegister">Write password</Label>
          <Input id="PasswordRegister" type="password" placeholder="Password" />
          <Label htmlFor="ConfirmPasswordRegister">Confirm your password</Label>
          <Input id="ConfirmPasswordRegister" type="password" placeholder="ConfirmPassword" />
        </Field>
      </CardContent>
      <CardFooter>
        <Field>
          <Button type="submit" className="w-full">
            Verify
          </Button>
        </Field>
      </CardFooter>
    </Card>
  )
}

export default StepPassword